package media

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rwcarlsen/goexif/exif"
)

type Jpg struct {
	Path            string
	fingerprint     string
	destinationPath LazyPath
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
	return j.destinationPath.GetDestinationPath(
		func() (string, error) {
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

			mediaHome := filepath.Join(base, string(photos), year, date, "sooc")
			// TODO: make this return just string and creation of directory is done somewhere else
			if _, err := os.Stat(mediaHome); os.IsNotExist(err) {
				err := os.MkdirAll(mediaHome, os.ModePerm)
				if err != nil {
					log.Fatalf("Error creating directory: %v", err)
				}
			}

			return filepath.Join(mediaHome, filepath.Base(j.Path)), nil
		})
}
