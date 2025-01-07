package workflow

import (
	"context"
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

type workerPool[T any] struct {
	jobs      chan T
	errorChan chan error
	wg        sync.WaitGroup
}

func newWorkerPool[T any](jobBufferSize int) *workerPool[T] {
	return &workerPool[T]{
		jobs:      make(chan T, jobBufferSize),
		errorChan: make(chan error, 1), // Buffer of 1 to ensure non-blocking
	}
}

func (wp *workerPool[T]) start(ctx context.Context, workerFunc func(T) error) {
	numWorkers := runtime.NumCPU() * 2

	for i := 0; i < numWorkers; i++ {
		wp.wg.Add(1)
		go func() {
			defer wp.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-wp.jobs:
					if !ok {
						return
					}
					if err := workerFunc(job); err != nil {
						select {
						case wp.errorChan <- err:
						default:
						}
					}
				}
			}
		}()
	}
}

func (wp *workerPool[T]) stop() {
	close(wp.jobs)
	wp.wg.Wait()
	close(wp.errorChan)
}

func (wp *workerPool[T]) enqueue(job T) {
	wp.jobs <- job
}

func (wp *workerPool[T]) errors() <-chan error {
	return wp.errorChan
}

type MediaMaps struct {
	SourceMap map[string]media.File
	DestMap   map[string][]media.File
}

func prepareMediaMaps(
	ctx context.Context,
	sourcePaths []string,
	destinationPath string,
	filter []string,
	noSooc bool,
) (MediaMaps, error) {
	var (
		sourceMedia      []media.File
		destinationMedia []media.File
	)

	for _, sourcePath := range sourcePaths {
		mediaFiles, err := scanFiles(ctx, sourcePath, filter, noSooc)
		if err != nil {
			return MediaMaps{}, fmt.Errorf("error occurred while scanning source directory '%s': %w", sourcePath, err)
		}
		sourceMedia = append(sourceMedia, mediaFiles...)
	}

	sourceMap := make(map[string]media.File)
	for _, mediaFile := range sourceMedia {
		fingerprint := mediaFile.GetFingerprint()
		if _, exists := sourceMap[fingerprint]; !exists {
			sourceMap[fingerprint] = mediaFile
		}
	}

	destinationMedia, err := scanFiles(ctx, destinationPath, filter, noSooc)
	if err != nil {
		return MediaMaps{}, fmt.Errorf("error occurred while scanning destination directory '%s': %w", destinationPath, err)
	}

	destMap := make(map[string][]media.File)
	for _, mediaFile := range destinationMedia {
		fingerprint := mediaFile.GetFingerprint()
		destMap[fingerprint] = append(destMap[fingerprint], mediaFile)
	}

	result := MediaMaps{
		SourceMap: sourceMap,
		DestMap:   destMap,
	}

	err = computeDestinationPaths(ctx, &result, destinationPath)
	if err != nil {
		return MediaMaps{}, err
	}

	return MediaMaps{
		SourceMap: sourceMap,
		DestMap:   destMap,
	}, nil
}

func computeDestinationPaths(ctx context.Context, mediaMaps *MediaMaps, dstPath string) error {
	wp := newWorkerPool[media.File](len(mediaMaps.SourceMap) + len(mediaMaps.DestMap))

	wp.start(ctx, func(file media.File) error {
		_, err := file.GetDestinationPath(dstPath)
		return err
	})

	go func() {
		defer wp.stop()
		for _, file := range mediaMaps.SourceMap {
			select {
			case <-ctx.Done():
				return
			default:
				wp.enqueue(file)
			}
		}
		for _, files := range mediaMaps.DestMap {
			for _, file := range files {
				select {
				case <-ctx.Done():
					return
				default:
					wp.enqueue(file)
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-wp.errors():
			if !ok {
				return nil
			}
			return fmt.Errorf("error occurred while computing destination path: %w", err)
		}
	}
}

func scanFiles(ctx context.Context, dirPath string, filter []string, noSooc bool) ([]media.File, error) {
	wp := newWorkerPool[string](100)
	resultsChan := make(chan media.File, 100)
	var results []media.File

	wp.start(ctx, func(path string) error {
		m, err := processFile(path, noSooc)
		if err != nil {
			return err
		}
		select {
		case resultsChan <- m:
		case <-ctx.Done():
			return context.Canceled
		}
		return nil
	})

	go func() {
		defer wp.stop()
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
			select {
			case <-ctx.Done():
				return context.Canceled
			default:
				wp.enqueue(path)
			}
			return nil
		})
		if err != nil {
			select {
			case wp.errorChan <- err:
			default:
			}
		}
	}()

	go func() {
		wp.wg.Wait()
		close(resultsChan)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err, ok := <-wp.errors():
			if !ok {
				return results, nil
			}
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

// Calculates the hash of the first and last chunks of a file.
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
