package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/rwcarlsen/goexif/exif"
)

type (
	MediaType  string
	ActionType string
)

const (
	Photos MediaType = "photos"
	Videos MediaType = "videos"

	ActionMove ActionType = "move"
	ActionSkip ActionType = "skip"

	appleEpochAdjustment = 2082844800

	movieResourceAtomType   = "moov"
	movieHeaderAtomType     = "mvhd"
	referenceMovieAtomType  = "rmra"
	compressedMovieAtomType = "cmov"

	OneKB = 1024
	OneMB = 1024 * OneKB
	OneGB = 1024 * OneMB
)

type args struct {
	Source      string `arg:"positional,required" help:"source directory for media"`
	Destination string `arg:"positional,required" help:"destination directory for orginised media"`
	DryRun      bool   `arg:"-d,--dryrun" default:"false" help:"does not modify file system"`
}

func (args) Description() string {
	return "Orginizes photo and video media into lightroom style directory structure"
}

type Action struct {
	Type           ActionType
	Source         Media
	Destination    Media
	DestinationDir string
}

func (a *Action) Execute() error {
	switch a.Type {
	case ActionMove:
		fmt.Printf("  Moving from %s to %s\n", a.Source.GetPath(), "unknown yet")
	case ActionSkip:
		fmt.Printf("  Skipping %s\n", a.Source.GetPath())
	default:
		panic(fmt.Errorf("unknown action type: %s", a.Type))
	}
	return nil
}

type Plan struct {
	Actions []Action
}

func (p *Plan) AddAction(action Action) {
	p.Actions = append(p.Actions, action)
}

func (p *Plan) Apply() error {
	fmt.Println("Applying plan:")

	for _, action := range p.Actions {
		err := action.Execute()
		if err != nil {
			return fmt.Errorf("error while executing action: %w", err)
		}
	}

	fmt.Printf("\n")
	return nil
}

func (p *Plan) PrintSummary() {
	copyCount := 0
	skipCount := 0

	fmt.Println("Detailed Actions:")
	for _, action := range p.Actions {
		switch action.Type {
		case ActionMove:
			fmt.Printf("  Move: %s\n", action.Source.GetPath())
			copyCount++
		case ActionSkip:
			fmt.Printf("  Skip: %s (already exists at %s)\n", action.Source.GetPath(), action.Destination.GetPath())
			skipCount++
		}
	}
	fmt.Printf("\n")

	fmt.Printf("Plan Summary:\n")
	fmt.Printf("  Files to move: %d\n", copyCount)
	fmt.Printf("  Files skipped: %d\n", skipCount)

	fmt.Printf("\n")
}

type Media interface {
	GetPath() string
	GetFingerprint() string
	SetFingerprint(fingerprint string)
	GetDestinationPath(base string) (string, error)
}

type Mov struct {
	Path        string
	Fingerprint string
}

func (m *Mov) GetPath() string {
	return m.Path
}

func (m *Mov) GetDestinationPath(base string) (string, error) {
	file, err := os.Open(m.Path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := make([]byte, 8)

	// Traverse videoBuffer to find movieResourceAtom
	for {
		// bytes 1-4 is atom size, 5-8 is type
		// Read atom
		if _, err := file.Read(buf); err != nil {
			return "", err
		}

		if bytes.Equal(buf[4:8], []byte(movieResourceAtomType)) {
			break // found it!
		}

		atomSize := binary.BigEndian.Uint32(buf) // check size of atom
		file.Seek(int64(atomSize)-8, 1)          // jump over data and set seeker at beginning of next atom
	}

	// read next atom
	if _, err := file.Read(buf); err != nil {
		return "", err
	}

	atomType := string(buf[4:8]) // skip size and read type
	switch atomType {
	case movieHeaderAtomType:
		// read next atom
		if _, err := file.Read(buf); err != nil {
			return "", err
		}

		// byte 1 is version, byte 2-4 is flags, 5-8 Creation time
		appleEpoch := int64(binary.BigEndian.Uint32(buf[4:])) // Read creation time

		creationTime := time.Unix(appleEpoch-appleEpochAdjustment, 0).Local()
		date := creationTime.Format("2006-01-02")
		year := strconv.Itoa(creationTime.Year())

		mediaHome := filepath.Join(base, string(Videos), year, date, "")
		if _, err := os.Stat(mediaHome); os.IsNotExist(err) {
			err := os.MkdirAll(mediaHome, os.ModePerm)
			if err != nil {
				log.Fatalf("Error creating directory: %v", err)
			}
		}

		return filepath.Join(mediaHome, filepath.Base(m.Path)), nil
	case compressedMovieAtomType:
		return "", errors.New("compressed video")
	case referenceMovieAtomType:
		return "", errors.New("reference video")
	default:
		return "", errors.New("did not find movie header atom (mvhd)")
	}
}

func (m *Mov) GetFingerprint() string {
	return m.Fingerprint
}

func (m *Mov) SetFingerprint(fingerprint string) {
	m.Fingerprint = fingerprint
}

type Jpg struct {
	Path        string
	Fingerprint string
}

func (j *Jpg) GetPath() string {
	return j.Path
}

func (j *Jpg) GetFingerprint() string {
	return j.Fingerprint
}

func (j *Jpg) SetFingerprint(fingerprint string) {
	j.Fingerprint = fingerprint
}

func (j *Jpg) GetMediaType() MediaType {
	return Photos
}

func (j *Jpg) GetDestinationPath(base string) (string, error) {
	f, err := os.Open(j.Path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	exif, err := exif.Decode(f)
	if err != nil {
		return "", err
	}

	creationTime, err := exif.DateTime()
	if err != nil {
		log.Fatal(err)
	}

	date := creationTime.Format("2006-01-02")
	year := strconv.Itoa(creationTime.Year())

	mediaHome := filepath.Join(base, string(Photos), year, date, "sooc")
	if _, err := os.Stat(mediaHome); os.IsNotExist(err) {
		err := os.MkdirAll(mediaHome, os.ModePerm)
		if err != nil {
			log.Fatalf("Error creating directory: %v", err)
		}
	}

	return filepath.Join(mediaHome, filepath.Base(j.Path)), nil
}

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

func (r *Raf) GetMediaType() MediaType {
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

// Smaller files get a fixed size; larger files use a percentage of the total size.
func calculateChunkSize(fileSize int64) int64 {
	const minChunkSize = OneMB
	const maxChunkSize = 10 * OneMB

	if fileSize < 100*OneMB { // Less than 100MB
		return minChunkSize
	}

	chunkSize := fileSize / 100 // 1% of the file size
	if chunkSize > maxChunkSize {
		return maxChunkSize
	}

	return chunkSize
}

// partialHash calculates the hash of the first and last chunks of a file.
func partialHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}
	fileSize := fileInfo.Size()
	chunkSize := calculateChunkSize(fileSize)

	hasher := sha256.New()
	buf := make([]byte, chunkSize)

	// Read the first chunk
	_, err = file.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read first chunk: %w", err)
	}
	hasher.Write(buf)

	// Seek to the last chunk
	if fileSize > chunkSize { // Only seek if the file is larger than the chunk size
		_, err = file.Seek(-chunkSize, io.SeekEnd)
		if err != nil {
			return "", fmt.Errorf("failed to seek to last chunk: %w", err)
		}

		_, err = file.Read(buf)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("failed to read last chunk: %w", err)
		}
		hasher.Write(buf)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func scanFiles(dirPath string) ([]Media, error) {
	var results []Media

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		var media Media

		switch ext {
		case ".jpg":
			media = &Jpg{Path: path}
		case ".raf":
			media = &Raf{Path: path}
		case ".mov":
			media = &Mov{Path: path}

		default:
			return fmt.Errorf("unsupported file: %s", path)
		}

		hash, err := partialHash(path)
		if err != nil {
			return fmt.Errorf("error calculating partial hash for %s: %w", path, err)
		}

		media.SetFingerprint(hash)
		results = append(results, media)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

func createPlan(sourcePath, destinationPath string) (Plan, error) {
	sourceMedia, err := scanFiles(sourcePath)
	if err != nil {
		return Plan{}, fmt.Errorf("error occured while scanning source directory: %w", err)
	}
	sourceMap := make(map[string]Media)
	for _, media := range sourceMedia {
		sourceMap[media.GetFingerprint()] = media
	}

	destinationMedia, err := scanFiles(destinationPath)
	if err != nil {
		return Plan{}, fmt.Errorf("error occured while scanning destination directory: %w", err)
	}
	destMap := make(map[string]Media)
	for _, media := range destinationMedia {
		destMap[media.GetFingerprint()] = media
	}

	plan := Plan{}

	// TODO: handle when media exists but is not orginized correctly so need to implement check for correct placement of destination media
	// media should determine it's own destinationPath then I can check correctness with current path
	// need to create a new loop for destiniation map and check if everything is placed correctly

	// for hash, destMedia := range destMap {
	// 	// check if path is correct
	//
	// 	// create error for duplicate media
	// }

	for hash, srcMedia := range sourceMap {
		if destMedia, exists := destMap[hash]; exists {
			plan.AddAction(Action{
				Type:           ActionSkip,
				Source:         srcMedia,
				Destination:    destMedia,
				DestinationDir: destinationPath,
			})
		} else {
			plan.AddAction(Action{
				Type:           ActionMove,
				Source:         srcMedia,
				DestinationDir: destinationPath,
			})
		}
	}

	plan.PrintSummary()

	return plan, nil
}

//
// func processFilesInDirectory(SourceDir string, destinationDir string, dryRun bool) {
// 	err := filepath.Walk(SourceDir, func(path string, info os.FileInfo, err error) error {
// 		if err != nil {
// 			return err
// 		}
//
// 		if info.IsDir() {
// 			return nil
// 		}
//
// 		ext := strings.ToLower(filepath.Ext(path))
// 		switch ext {
// 		case ".jpg":
// 			jpg := &Jpg{Path: path}
//
// 			exif, err := jpg.ReadExif()
// 			if err != nil {
// 				log.Fatal(err)
// 			}
// 			fmt.Println(exif)
//
// 			dateTime, err := exif.DateTime()
// 			if err != nil {
// 				log.Fatal(err)
// 			}
//
// 			mediaHome := createMediaDir(destinationDir, Photos, dateTime, "sooc", dryRun)
// 			destPath := filepath.Join(mediaHome, filepath.Base(path))
// 			movefile(path, destPath, dryRun)
// 		case ".raf":
// 			raf := &Raf{Path: path}
//
// 			exif, err := raf.ReadExif()
// 			if err != nil {
// 				log.Fatal(err)
// 			}
//
// 			dateTime, err := exif.DateTime()
// 			if err != nil {
// 				log.Fatal(err)
// 			}
//
// 			mediaHome := createMediaDir(destinationDir, Photos, dateTime, "", dryRun)
// 			destPath := filepath.Join(mediaHome, filepath.Base(path))
// 			movefile(path, destPath, dryRun)
// 		case ".mov":
// 			mov := &Mov{Path: path}
//
// 			creationTime, err := mov.ReadCreationTime()
// 			if err != nil {
// 				log.Fatal(err)
// 			}
//
// 			mediaHome := createMediaDir(destinationDir, Videos, creationTime, "", dryRun)
// 			destPath := filepath.Join(mediaHome, filepath.Base(path))
// 			movefile(path, destPath, dryRun)
// 		default:
// 			log.Fatalf("unsupported file: %s\n", path)
// 		}
//
// 		return nil
// 	})
// 	if err != nil {
// 		log.Fatalf("Error reading directory: %v", err)
// 	}
// }
//
// func createMediaDir(destinationDir string, mediaType MediaType, dateTime time.Time, subPath string, dryRun bool) string {
// 	date := dateTime.Format("2006-01-02")
// 	year := strconv.Itoa(dateTime.Year())
// 	path := filepath.Join(destinationDir, string(mediaType), year, date, subPath)
//
// 	if dryRun {
// 		fmt.Printf("creating directory: %s", path)
// 	} else {
// 		if _, err := os.Stat(path); os.IsNotExist(err) {
// 			err := os.MkdirAll(path, os.ModePerm)
// 			if err != nil {
// 				log.Fatalf("Error creating directory: %v", err)
// 			}
// 		}
// 	}
//
// 	return path
// }
//
// func movefile(source string, destination string, dryRun bool) {
// 	if dryRun {
// 		fmt.Printf("moving %s to %s", source, destination)
// 	} else {
// 		err := os.Rename(source, destination)
// 		if err != nil {
// 			log.Fatalf("error moving file: %v", err)
// 		}
// 	}
// }

func run() error {
	var args args
	arg.MustParse(&args)

	plan, err := createPlan(args.Source, args.Destination)
	if err != nil {
		return err
	}

	if !args.DryRun {
		plan.Apply()
	}

	return nil
	// processFilesInDirectory(args.Source, args.Destination, args.DryRun)
	// BUG: _embeded jpg gets created next to raf files. don't do that
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// TODO: create plan before making changes making sure there are no conflicts and create a report with changes this will be either --plan or --dry-run
// TODO: never overwrite existing files
// TODO: clean up source dir if it's empty of content

// Scan Files:
//
//     For each file, compute its partial hash and include metadata.
//
// Check Uniqueness:
//
//     Compare the computed hash to the hashes of files in the destination folder.
//     Since you're rehashing for every run, the destination folder itself serves as the "state."
//
// Resolve Conflicts:
//
//     If a computed hash matches a file already in the destination folder:
//         Skip the file (if content is identical).
//         Log an error or handle the conflict (if content differs).
//
// Simulate (Dry Run):
//
//     Before making changes, output a plan of what will happen (e.g., files to copy, skip, or rename).
//
// Execute Plan:
//
//     Perform the copy/move operations.
//
//
//
//
