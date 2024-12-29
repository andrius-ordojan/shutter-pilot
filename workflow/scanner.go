package workflow

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/andrius-ordojan/shutter-pilot/media"
)

// TODO: add context.Context to be able shutdown all threads safly from terminal kill command
func scanFiles(dirPath string, filter []string, noSooc bool) ([]media.File, error) {
	var results []media.File

	jobs := make(chan string, 100)
	resultsChan := make(chan media.File, 100)
	errorChan := make(chan error, 1) // Buffer of 1 to ensure non-blocking

	var wg sync.WaitGroup

	numWorkers := runtime.NumCPU() * 2
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				m, err := processFile(path, noSooc)
				if err != nil {
					select {
					case errorChan <- err:
					default:
					}
					return
				}
				resultsChan <- m
			}
		}()
	}

	// TODO: Instead of sending individual file paths to the jobs channel, send batches of file paths
	go func() {
		defer close(jobs)
		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			filetype := strings.TrimPrefix(ext, ".")
			if !slices.Contains(filter, filetype) {
				return nil
			}

			jobs <- path

			return nil
		})
		if err != nil {
			select {
			case errorChan <- err:
			default:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for {
		select {
		case err := <-errorChan:
			return nil, err
		case m, ok := <-resultsChan:
			if !ok {
				return results, nil
			}
			results = append(results, m)
		}
	}
}

func processFile(path string, noSooc bool) (media.File, error) {
	ext := strings.ToLower(filepath.Ext(path))
	filetype := strings.TrimPrefix(ext, ".")

	var m media.File
	switch media.MediaType(filetype) {
	case media.JpgMedia:
		m = media.NewJpg(path, noSooc)
	case media.RafMedia:
		m = media.NewRaf(path)
	case media.MovMedia:
		m = media.NewMov(path)
	default:
		return nil, fmt.Errorf("unsupported media type: %s", path)
	}

	hash, err := partialHash(path)
	if err != nil {
		return nil, fmt.Errorf("error calculating partial hash for %s: %w", path, err)
	}

	m.SetFingerprint(hash)
	return m, nil
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
