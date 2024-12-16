package media

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rwcarlsen/goexif/exif"
)

type Raf struct {
	Path        string
	Fingerprint string

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

func (r *Raf) GetPath() string {
	return r.Path
}

func (r *Raf) GetFingerprint() string {
	return r.Fingerprint
}

func (r *Raf) SetFingerprint(fingerprint string) {
	r.Fingerprint = fingerprint
}

func (r *Raf) GetMediaType() Type {
	return Photos
}

func (r *Raf) GetDestinationPath(base string) (string, error) {
	f, err := os.Open(r.Path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	err = binary.Read(f, binary.BigEndian, &r.Header)
	if err != nil {
		return "", err
	}

	jbuf := make([]byte, r.Header.Dir.Jpeg.Len)
	_, err = f.ReadAt(jbuf, int64(r.Header.Dir.Jpeg.Idx))
	if err != nil {
		return "", err
	}
	r.Jpeg = jbuf

	buf := bytes.NewBuffer(jbuf)
	r.Exif, err = exif.Decode(buf)
	if err != nil {
		return "", err
	}

	creationTime, err := r.Exif.DateTime()
	if err != nil {
		log.Fatal(err)
	}

	date := creationTime.Format("2006-01-02")
	year := strconv.Itoa(creationTime.Year())

	mediaHome := filepath.Join(base, string(Photos), year, date, "")
	if _, err := os.Stat(mediaHome); os.IsNotExist(err) {
		err := os.MkdirAll(mediaHome, os.ModePerm)
		if err != nil {
			log.Fatalf("Error creating directory: %v", err)
		}
	}

	return filepath.Join(mediaHome, filepath.Base(r.Path)), nil
}
