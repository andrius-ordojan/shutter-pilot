package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
)

type (
	TestMediaType string
)

const (
	JpgFile TestMediaType = "jpg"
	RafFile TestMediaType = "raf"
	MovFile TestMediaType = "mov"
)

type TestMediaFile struct {
	Name                string
	Type                TestMediaType
	SourceDir           string
	DestinationDir      string
	ExpectedDestination string
}

func (m *TestMediaFile) CopyTo(destination string) error {
	if m.Name == "" {
		panic("name is not set")
	}

	src := filepath.Join("./testdata/", m.Name)
	dest := filepath.Join(destination, m.Name)

	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	err = os.MkdirAll(destination, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination directories: %w", err)
	}

	destinationFile, err := os.Create(dest)
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

func (m *TestMediaFile) FullExpectedDestination() string {
	if m.DestinationDir == "" {
		panic("destination dir is not set")
	}
	if m.ExpectedDestination == "" {
		panic("expected destination is not set")
	}
	return filepath.Join(m.DestinationDir, m.ExpectedDestination)
}

func (m *TestMediaFile) CopyToExpectedDestination() error {
	err := m.CopyTo(m.FullExpectedDestination())
	if err != nil {
		return err
	}
	return nil
}

func (m *TestMediaFile) CheckExistsAtExpectedDestination() error {
	path := m.FullExpectedDestination()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("media expected at: %s, but not found", path)
	} else if err != nil {
		return fmt.Errorf("error checking file %s: %w", path, err)
	}
	return nil
}

var testMediaFiles = []*TestMediaFile{
	{Name: "DSCF9533.RAF", Type: RafFile, ExpectedDestination: "photos/2024/2024-12-07"},
	{Name: "DSCF9533.JPG", Type: JpgFile, ExpectedDestination: "photos/2024/2024-12-07/sooc"},
	{Name: "DSCF3517.JPG", Type: JpgFile, ExpectedDestination: "photos/2024/2024-11-13/sooc"},
	{Name: "DSCF3517.RAF", Type: RafFile, ExpectedDestination: "photos/2024/2024-11-13"},
	{Name: "DSCF9531.MOV", Type: MovFile, ExpectedDestination: "videos/2024/2024-12-07"},
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

	for _, m := range testMediaFiles {
		m.SourceDir = sourceDir
		m.DestinationDir = destDir
		m.CopyTo(sourceDir)
		m.CopyToExpectedDestination()
	}

	suppressOutput(func() {
		setArgs(t, "app", sourceDir, destDir)
		err = run()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, m := range testMediaFiles {
		err := m.CheckExistsAtExpectedDestination()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func Test_ShouldMove_WhenMediaDoesNotExist(t *testing.T) {
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

	// TODO: make a function to help with skipping some numebr of media
	jpgCount := 0
	rafCount := 0
	movCount := 0
	for _, m := range testMediaFiles {
		m.SourceDir = sourceDir
		m.DestinationDir = destDir
		m.CopyTo(sourceDir)

		switch m.Type {
		case JpgFile:
			if jpgCount > 0 {
				m.CopyToExpectedDestination()
			}
			jpgCount++
		case RafFile:
			if rafCount > 0 {
				m.CopyToExpectedDestination()
			}
			rafCount++
		case MovFile:
			if movCount > 0 {
				m.CopyToExpectedDestination()
			}
			movCount++
		default:
			panic("Type is not supported")
		}
	}

	setArgs(t, "app", sourceDir, destDir)
	err = run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// TODO: uncomment later
	// suppressOutput(func() {
	// 	setArgs(t, "app", sourceDir, destDir)
	// 	results, err = run()
	// 	if err != nil {
	// 		t.Fatalf("unexpected error: %v", err)
	// 	}
	// })

	// TODO: make helper function to assist in checking all media
	for _, m := range testMediaFiles {
		err := m.CheckExistsAtExpectedDestination()
		if err != nil {
			t.Fatal(err)
		}
	}
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
// test what happens when unsupported media is present
