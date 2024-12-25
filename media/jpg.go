package media

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rwcarlsen/goexif/exif"
)

func NewJpg(path string, noSooc bool) *Jpg {
	if path == "" {
		panic("path not set for media file")
	}

	return &Jpg{Path: path, noSooc: noSooc}
}

type Jpg struct {
	Path        string
	fingerprint string
	lazy        LazyPath
	noSooc      bool
}

func (j *Jpg) GetPath() string {
	return j.Path
}

func (j *Jpg) GetFingerprint() string {
	return j.fingerprint
}

func (j *Jpg) SetFingerprint(fingerprint string) {
	j.fingerprint = fingerprint
}

func (j *Jpg) GetDestinationPath(base string) (string, error) {
	return j.lazy.GetDestinationPath(
		func() (string, error) {
			f, err := os.Open(j.Path)
			if err != nil {
				return "", err
			}
			defer f.Close()

			exif, err := exif.Decode(f)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return "", errors.New("exif data not found")
				} else {
					return "", fmt.Errorf("failed to decode exif data: %w", err)
				}
			}

			creationTime, err := exif.DateTime()
			if err != nil {
				return "", fmt.Errorf("failed to get creation time: %w", err)
			}

			date := creationTime.Format("2006-01-02")
			year := strconv.Itoa(creationTime.Year())

			subFolder := "sooc"
			if j.noSooc {
				subFolder = ""
			}
			mediaHome := filepath.Join(base, string(photos), year, date, subFolder)
			return filepath.Join(mediaHome, filepath.Base(j.Path)), nil
		})
}
