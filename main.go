package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/rwcarlsen/goexif/exif"
)

type MediaType string

const (
	Photos MediaType = "photos"
	Videos MediaType = "videos"

	appleEpochAdjustment = 2082844800

	movieResourceAtomType   = "moov"
	movieHeaderAtomType     = "mvhd"
	referenceMovieAtomType  = "rmra"
	compressedMovieAtomType = "cmov"
)

type Mov struct {
	FilePath string
}

func (mov *Mov) ReadCreationTime() (time.Time, error) {
	file, err := os.Open(mov.FilePath)
	if err != nil {
		return time.Time{}, err
	}
	defer file.Close()

	buf := make([]byte, 8)

	// Traverse videoBuffer to find movieResourceAtom
	for {
		// bytes 1-4 is atom size, 5-8 is type
		// Read atom
		if _, err := file.Read(buf); err != nil {
			return time.Time{}, err
		}

		if bytes.Equal(buf[4:8], []byte(movieResourceAtomType)) {
			break // found it!
		}

		atomSize := binary.BigEndian.Uint32(buf) // check size of atom
		file.Seek(int64(atomSize)-8, 1)          // jump over data and set seeker at beginning of next atom
	}

	// read next atom
	if _, err := file.Read(buf); err != nil {
		return time.Time{}, err
	}

	atomType := string(buf[4:8]) // skip size and read type
	switch atomType {
	case movieHeaderAtomType:
		// read next atom
		if _, err := file.Read(buf); err != nil {
			return time.Time{}, err
		}

		// byte 1 is version, byte 2-4 is flags, 5-8 Creation time
		appleEpoch := int64(binary.BigEndian.Uint32(buf[4:])) // Read creation time

		return time.Unix(appleEpoch-appleEpochAdjustment, 0).Local(), nil
	case compressedMovieAtomType:
		return time.Time{}, errors.New("compressed video")
	case referenceMovieAtomType:
		return time.Time{}, errors.New("reference video")
	default:
		return time.Time{}, errors.New("did not find movie header atom (mvhd)")
	}
}

type Jpg struct {
	FilePath string
}

func (jpg *Jpg) ReadExif() (*exif.Exif, error) {
	f, err := os.Open(jpg.FilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	exif, err := exif.Decode(f)
	if err != nil {
		return nil, err
	}

	return exif, nil
}

type Raf struct {
	FilePath string

	Header struct {
		Magic         [16]byte
		FormatVersion [4]byte
		CameraId      [8]byte
		Camera        [32]byte
		Dir           struct {
			Version [4]byte
			_       [20]byte
			Jpeg    struct {
				Idx int32
				Len int32
			}
			CfaHeader struct {
				Idx int32
				Len int32
			}
			Cfa struct {
				Idx int32
				Len int32
			}
		}
	}
	Jpeg []byte
	Exif *exif.Exif
}

func (raf *Raf) ReadExif() (*exif.Exif, error) {
	f, err := os.Open(raf.FilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	err = binary.Read(f, binary.BigEndian, &raf.Header)
	if err != nil {
		return nil, err
	}

	jbuf := make([]byte, raf.Header.Dir.Jpeg.Len)
	_, err = f.ReadAt(jbuf, int64(raf.Header.Dir.Jpeg.Idx))
	if err != nil {
		return nil, err
	}
	raf.Jpeg = jbuf

	buf := bytes.NewBuffer(jbuf)
	raf.Exif, err = exif.Decode(buf)
	if err != nil {
		return nil, err
	}

	return raf.Exif, nil
}

func processFilesInDirectory(SourceDir string, destinationDir string, dryRun bool) {
	err := filepath.Walk(SourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".jpg":
			jpg := &Jpg{FilePath: path}

			exif, err := jpg.ReadExif()
			if err != nil {
				log.Fatal(err)
			}

			dateTime, err := exif.DateTime()
			if err != nil {
				log.Fatal(err)
			}

			mediaHome := createMediaDir(destinationDir, Photos, dateTime, "sooc", dryRun)
			destPath := filepath.Join(mediaHome, filepath.Base(path))
			movefile(path, destPath, dryRun)
		case ".raf":
			raf := &Raf{FilePath: path}

			exif, err := raf.ReadExif()
			if err != nil {
				log.Fatal(err)
			}

			dateTime, err := exif.DateTime()
			if err != nil {
				log.Fatal(err)
			}

			mediaHome := createMediaDir(destinationDir, Photos, dateTime, "", dryRun)
			destPath := filepath.Join(mediaHome, filepath.Base(path))
			movefile(path, destPath, dryRun)
		case ".mov":
			mov := &Mov{FilePath: path}

			creationTime, err := mov.ReadCreationTime()
			if err != nil {
				log.Fatal(err)
			}

			mediaHome := createMediaDir(destinationDir, Videos, creationTime, "", dryRun)
			destPath := filepath.Join(mediaHome, filepath.Base(path))
			movefile(path, destPath, dryRun)
		default:
			log.Fatalf("unsupported file: %s\n", path)
		}

		return nil
	})
	if err != nil {
		log.Fatalf("Error reading directory: %v", err)
	}
}

func createMediaDir(destinationDir string, mediaType MediaType, dateTime time.Time, subPath string, dryRun bool) string {
	date := dateTime.Format("2006-01-02")
	year := strconv.Itoa(dateTime.Year())
	path := filepath.Join(destinationDir, string(mediaType), year, date, subPath)

	if dryRun {
		fmt.Printf("creating directory: %s", path)
	} else {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			err := os.MkdirAll(path, os.ModePerm)
			if err != nil {
				log.Fatalf("Error creating directory: %v", err)
			}
		}
	}

	return path
}

func movefile(source string, destination string, dryRun bool) {
	if dryRun {
		fmt.Printf("moving %s to %s", source, destination)
	} else {
		err := os.Rename(source, destination)
		if err != nil {
			log.Fatalf("error moving file: %v", err)
		}
	}
}

type args struct {
	Source      string `arg:"positional,required" help:"source directory for media"`
	Destination string `arg:"positional,required" help:"destination directory for orginised media"`
	DryRun      bool   `arg:"-d,--dryrun" default:"false" help:"does not modify file system"`
}

func (args) Description() string {
	return "Orginizes photo and video media into lightroom style directory structure"
}

func main() {
	var args args
	arg.MustParse(&args)

	// TODO: handle duplicates by not overriding them
	processFilesInDirectory(args.Source, args.Destination, args.DryRun)

	// TODO: clean up source dir if it's empty of content
}
