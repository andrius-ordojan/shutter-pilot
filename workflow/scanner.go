package workflow

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/andrius-ordojan/shutter-pilot/media"
)

func scanFiles(dirPath string) ([]media.File, error) {
	var results []media.File

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		var m media.File
		switch ext {
		case ".jpg":
			m = &media.Jpg{Path: path}
		case ".raf":
			m = &media.Raf{Path: path}
		case ".mov":
			m = &media.Mov{Path: path}
		default:
			return fmt.Errorf("unsupported file: %s", path)
		}

		hash, err := partialHash(path)
		if err != nil {
			return fmt.Errorf("error calculating partial hash for %s: %w", path, err)
		}

		m.SetFingerprint(hash)
		results = append(results, m)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

func calculateChunkSize(fileSize int64) int64 {
	const minChunkSize = oneMB
	const maxChunkSize = 10 * oneMB

	if fileSize < 100*oneMB {
		return minChunkSize
	}

	chunkSize := fileSize / 100
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
