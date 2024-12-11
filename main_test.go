package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

const (
	testDataDir = "./testdata/"
)

type TestMediaFile struct {
	Name         string
	CreationDate string
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy content: %w", err)
	}

	err = destinationFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	return nil
}

func setArgs(t *testing.T, args ...string) {
	originalArgs := os.Args

	os.Args = args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
}

func TestIntegration_ShouldSkip_WhenMediaExists(t *testing.T) {
	testSourceDir, err := os.MkdirTemp(".", "tmptest")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(testSourceDir)

	media := &TestMediaFile{Name: "DSCF9533.RAF", CreationDate: "2024-12-07"}
	copyFile(filepath.Join(testDataDir, media.Name), filepath.Join(testSourceDir, media.Name))
	media = &TestMediaFile{Name: "DSCF9533.JPG", CreationDate: "2024-12-07"}
	copyFile(filepath.Join(testDataDir, media.Name), filepath.Join(testSourceDir, media.Name))
	media = &TestMediaFile{Name: "DSCF3517.JPG", CreationDate: "2024-11-13"}
	copyFile(filepath.Join(testDataDir, media.Name), filepath.Join(testSourceDir, media.Name))
	media = &TestMediaFile{Name: "DSCF3517.RAF", CreationDate: "2024-11-13"}
	copyFile(filepath.Join(testDataDir, media.Name), filepath.Join(testSourceDir, media.Name))
	media = &TestMediaFile{Name: "DSCF9531.MOV", CreationDate: "2024-12-07"}
	copyFile(filepath.Join(testDataDir, media.Name), filepath.Join(testSourceDir, media.Name))

	setArgs(t, "app", testSourceDir, "test-output")
	run()
}

func TestShouldMoveWhenMediaDoesNotExist(t *testing.T) {
}

func TestShouldProcessFilesWhenTheyAreLocatedInSubfolders(t *testing.T) {
}

func TestShouldMoveToSoocDirWhenProcessingJpgMedia(t *testing.T) {
}

func TestShouldMoveToPhotosDirWhenProcessingJpgOrRafMedia(t *testing.T) {
}

func TestShouldMoveToVideosDirWhenProcessingMovMedia(t *testing.T) {
}

func TestShouldSkipWhenMediaIsCopyButNameIsDifferent(t *testing.T) {
}

func TestShouldMoveWhenMediaExistsButIsNotPlacedCorrectly(t *testing.T) {
}

func TestShouldErrorWhenMetadataNotPresent(t *testing.T) {
}

func TestIntegration_ShouldError_WhenDestinationFolderDoesNotExist(t *testing.T) {
	sourceDir, err := os.MkdirTemp(".", "tmptest")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(sourceDir)

	setArgs(t, "app", sourceDir, "notExist")
	err = run()

	if err == nil {
		t.Fatalf("expected an error, but got nil")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected error to be os.ErrNotExist, but got: %v", err)
	}
}

func TestIntegration_ShouldError_WhenSourceFolderDoesNotExist(t *testing.T) {
	destDir, err := os.MkdirTemp(".", "tmptest")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(destDir)

	setArgs(t, "app", "notExist", destDir)
	err = run()

	if err == nil {
		t.Fatalf("expected an error, but got nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected error to be os.ErrNotExist, but got: %v", err)
	}
}

// TODO: test dynamic hash chunk using mocking
