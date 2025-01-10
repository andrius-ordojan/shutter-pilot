package workflow

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/andrius-ordojan/shutter-pilot/media"
)

type progressReport struct {
	Processed int64
	Total     int64
}

type workerPool[T any] struct {
	jobs          chan T
	errorChan     chan error
	progressChan  chan progressReport
	totalJobs     atomic.Int64
	processedJobs atomic.Int64
	workerWG      sync.WaitGroup
	reporterWG    sync.WaitGroup
}

func newWorkerPool[T any](jobBufferSize int) *workerPool[T] {
	return &workerPool[T]{
		jobs:         make(chan T, jobBufferSize),
		errorChan:    make(chan error, 1), // Buffer of 1 to ensure non-blocking
		progressChan: make(chan progressReport, 100),
	}
}

func (wp *workerPool[T]) start(ctx context.Context, workerFunc func(T) error) {
	numWorkers := runtime.NumCPU() * 2

	wp.reporterWG.Add(1)
	go wp.startProgressReporter(ctx)

	for i := 0; i < numWorkers; i++ {
		wp.workerWG.Add(1)
		go func() {
			defer wp.workerWG.Done()
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
					wp.processedJobs.Add(1)
					wp.sendProgress()
				}
			}
		}()
	}
}

func (wp *workerPool[T]) startProgressReporter(ctx context.Context) {
	defer wp.reporterWG.Done()

	const progressStep = 20.0
	var lastReportedPercentage float64

	for {
		select {
		case <-ctx.Done():
			return
		case progress, ok := <-wp.progressChan:
			if !ok {
				return
			}

			if progress.Total == 0 {
				return
			}

			currentPercentage := (float64(progress.Processed) / float64(progress.Total)) * 100
			if currentPercentage >= lastReportedPercentage+progressStep || currentPercentage == 100 {
				fmt.Printf("    Processed %d/%d files (%.0f%%)\n", progress.Processed, progress.Total, currentPercentage)
				lastReportedPercentage = currentPercentage - math.Mod(currentPercentage, progressStep)
			}

			if progress.Processed == progress.Total {
				return
			}
		}
	}
}

func (wp *workerPool[T]) sendProgress() {
	progress := progressReport{
		Processed: wp.processedJobs.Load(),
		Total:     wp.totalJobs.Load(),
	}
	wp.progressChan <- progress
}

func (wp *workerPool[T]) stop(optionalFunc func()) {
	close(wp.jobs)
	wp.workerWG.Wait()

	wp.sendProgress()
	wp.reporterWG.Wait()

	close(wp.errorChan)
	close(wp.progressChan)

	if optionalFunc != nil {
		optionalFunc()
	}
}

func (wp *workerPool[T]) enqueue(job T) {
	wp.jobs <- job
	wp.totalJobs.Add(1)
	wp.sendProgress()
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

	fmt.Println()

	result := MediaMaps{
		SourceMap: sourceMap,
		DestMap:   destMap,
	}
	err = computeDestinationPaths(ctx, &result, destinationPath)
	if err != nil {
		return MediaMaps{}, err
	}

	fmt.Println()

	return MediaMaps{
		SourceMap: sourceMap,
		DestMap:   destMap,
	}, nil
}

func computeDestinationPaths(ctx context.Context, mediaMaps *MediaMaps, dstPath string) error {
	destLen := 0
	for _, files := range mediaMaps.DestMap {
		destLen += len(files)
	}
	bufferLen := len(mediaMaps.SourceMap) + destLen
	wp := newWorkerPool[media.File](bufferLen)

	for _, file := range mediaMaps.SourceMap {
		select {
		case <-ctx.Done():
			return nil
		default:
			wp.enqueue(file)
		}
	}
	for _, files := range mediaMaps.DestMap {
		for _, file := range files {
			select {
			case <-ctx.Done():
				return nil
			default:
				wp.enqueue(file)
			}
		}
	}

	fmt.Printf("  calculating destinations for %d files\n", wp.totalJobs.Load())

	wp.start(ctx, func(file media.File) error {
		_, err := file.GetDestinationPath(dstPath)
		return err
	})
	go wp.stop(nil)

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
	resultsChan := make(chan media.File, 100)
	var results []media.File

	wp := newWorkerPool[string](100)

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

	fmt.Printf("  scanning %s: %d files\n", dirPath, wp.totalJobs.Load())

	wp.start(ctx, func(path string) error {
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
			return fmt.Errorf("unsupported media type: %s", path)
		}

		hash, err := partialHash(path)
		if err != nil {
			return fmt.Errorf("error calculating partial hash for %s: %w", path, err)
		}

		m.SetFingerprint(hash)

		select {
		case resultsChan <- m:
		case <-ctx.Done():
			return context.Canceled
		}

		return nil
	})

	go wp.stop(func() {
		close(resultsChan)
	})

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
