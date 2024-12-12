package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
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

func ls(dir string) {
	fmt.Printf("ls: %s\n", dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	for _, e := range entries {
		fmt.Println(e.Name())
	}
}

func suppressOutput(f func()) {
	// Save the original stdout and stderr
	originalStdout := os.Stdout
	originalStderr := os.Stderr

	// Redirect stdout and stderr to /dev/null
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		panic("failed to open /dev/null")
	}
	defer devNull.Close()

	os.Stdout = devNull
	os.Stderr = devNull

	// Run the function
	defer func() {
		os.Stdout = originalStdout
		os.Stderr = originalStderr
	}()
	f()
}

func Test_ShouldSkip_WhenMediaExists(t *testing.T) {
	sourceDir, err := os.MkdirTemp(".", "tmp_source")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(sourceDir)

	destDir, err := os.MkdirTemp(".", "tmp_dest")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(destDir)

	media := &TestMediaFile{Name: "DSCF9533.RAF", CreationDate: "2024-12-07"}
	copyFile(filepath.Join(testDataDir, media.Name), filepath.Join(sourceDir, media.Name))
	copyFile(filepath.Join(testDataDir, media.Name), filepath.Join(destDir, media.Name))
	media = &TestMediaFile{Name: "DSCF9533.JPG", CreationDate: "2024-12-07"}
	copyFile(filepath.Join(testDataDir, media.Name), filepath.Join(sourceDir, media.Name))
	copyFile(filepath.Join(testDataDir, media.Name), filepath.Join(destDir, media.Name))
	media = &TestMediaFile{Name: "DSCF3517.JPG", CreationDate: "2024-11-13"}
	copyFile(filepath.Join(testDataDir, media.Name), filepath.Join(sourceDir, media.Name))
	copyFile(filepath.Join(testDataDir, media.Name), filepath.Join(destDir, media.Name))
	media = &TestMediaFile{Name: "DSCF3517.RAF", CreationDate: "2024-11-13"}
	copyFile(filepath.Join(testDataDir, media.Name), filepath.Join(sourceDir, media.Name))
	copyFile(filepath.Join(testDataDir, media.Name), filepath.Join(destDir, media.Name))
	media = &TestMediaFile{Name: "DSCF9531.MOV", CreationDate: "2024-12-07"}
	copyFile(filepath.Join(testDataDir, media.Name), filepath.Join(sourceDir, media.Name))
	copyFile(filepath.Join(testDataDir, media.Name), filepath.Join(destDir, media.Name))

	var results []string
	suppressOutput(func() {
		setArgs(t, "app", sourceDir, destDir)
		results, err = run()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, r := range results {
		if !strings.Contains(r, string(ActionSkip)) {
			t.Error("expected to only have skip actions")
		}
	}
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
	_, err = run()

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
	_, err = run()

	if err == nil {
		t.Fatalf("expected an error, but got nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected error to be os.ErrNotExist, but got: %v", err)
	}
}

// TODO: test dynamic hash chunk using mocking
