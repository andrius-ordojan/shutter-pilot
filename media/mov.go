package media

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	appleEpochAdjustment = 2082844800

	movieResourceAtomType   = "moov"
	movieHeaderAtomType     = "mvhd"
	referenceMovieAtomType  = "rmra"
	compressedMovieAtomType = "cmov"
)

func NewMov(path string) *Mov {
	if path == "" {
		panic("path not set for media file")
	}

	return &Mov{Path: path}
}

type Mov struct {
	Path        string
	fingerprint string
	lazy        LazyPath
}

func (m *Mov) GetPath() string {
	return m.Path
}

func (m *Mov) GetDestinationPath(base string) (string, error) {
	return m.lazy.GetDestinationPath(
		func() (string, error) {
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
				if atomSize < 8 {
					return "", errors.New("invalid atom size")
				}
				file.Seek(int64(atomSize)-8, 1) // jump over data and set seeker at beginning of next atom
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

				creationTimeValue := binary.BigEndian.Uint32(buf[4:])
				if creationTimeValue == 0 {
					return "", errors.New("creation time not set in metadata")
				}
				// byte 1 is version, byte 2-4 is flags, 5-8 Creation time
				appleEpoch := int64(creationTimeValue) // Read creation time

				creationTime := time.Unix(appleEpoch-appleEpochAdjustment, 0).Local()
				date := creationTime.Format("2006-01-02")
				year := strconv.Itoa(creationTime.Year())

				mediaHome := filepath.Join(base, string(videos), year, date)
				return filepath.Join(mediaHome, filepath.Base(m.Path)), nil
			case compressedMovieAtomType:
				return "", errors.New("compressed video")
			case referenceMovieAtomType:
				return "", errors.New("reference video")
			default:
				return "", errors.New("did not find movie header atom (mvhd)")
			}
		})
}

func (m *Mov) GetFingerprint() string {
	return m.fingerprint
}

func (m *Mov) SetFingerprint(fingerprint string) {
	m.fingerprint = fingerprint
}
