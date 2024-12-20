package media

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rwcarlsen/goexif/exif"
)

type Raf struct {
	Path            string
	fingerprint     string
	destinationPath LazyPath

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
}

func (r *Raf) GetPath() string {
	return r.Path
}

func (r *Raf) GetFingerprint() string {
	return r.fingerprint
}

func (r *Raf) SetFingerprint(fingerprint string) {
	r.fingerprint = fingerprint
}

func (r *Raf) GetDestinationPath(base string) (string, error) {
	return r.destinationPath.GetDestinationPath(
		func() (string, error) {
			f, err := os.Open(r.Path)
			if err != nil {
				return "", err
			}
			defer f.Close()

			err = binary.Read(f, binary.BigEndian, &r.Header)
			if err != nil {
				return "", fmt.Errorf("failed to read RAF header: %w", err)
			}

			jbuf := make([]byte, r.Header.Dir.Jpeg.Len)
			_, err = f.ReadAt(jbuf, int64(r.Header.Dir.Jpeg.Idx))
			if err != nil {
				return "", fmt.Errorf("failed to read JPEG data: %w", err)
			}
			exifData, err := exif.Decode(bytes.NewReader(jbuf))
			if err != nil {
				return "", fmt.Errorf("failed to decode EXIF data: %w", err)
			}

			creationTime, err := exifData.DateTime()
			if err != nil {
				return "", fmt.Errorf("failed to get creation time: %w", err)
			}

			date := creationTime.Format("2006-01-02")
			year := strconv.Itoa(creationTime.Year())

			mediaHome := filepath.Join(base, string(photos), year, date, "")
			return filepath.Join(mediaHome, filepath.Base(r.Path)), nil
		})
}
