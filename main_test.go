package main

import (
	"errors"
	"fmt"
	"io"
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

func (m *TestMediaFile) CheckExistsAt(path string) error {
	fullPath := filepath.Join(path, m.Name)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("media expected at: %s, but not found", fullPath)
	} else if err != nil {
		return fmt.Errorf("error checking file %s: %w", fullPath, err)
	}
	return nil
}

func (m *TestMediaFile) CheckMissingAt(path string) error {
	fullPath := filepath.Join(path, m.Name)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("error checking file %s: %w", fullPath, err)
	}
	return fmt.Errorf("media not expected at: %s, but found", fullPath)
}

var testMediaFiles = []*TestMediaFile{
	{Name: "DSCF9533.RAF", Type: RafFile, ExpectedDestination: "photos/2024/2024-12-07"},
	{Name: "DSCF9533.JPG", Type: JpgFile, ExpectedDestination: "photos/2024/2024-12-07/sooc"},
	{Name: "DSCF3517.JPG", Type: JpgFile, ExpectedDestination: "photos/2024/2024-11-13/sooc"},
	{Name: "DSCF3517.RAF", Type: RafFile, ExpectedDestination: "photos/2024/2024-11-13"},
	{Name: "DSCF9531.MOV", Type: MovFile, ExpectedDestination: "videos/2024/2024-12-07"},
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

	err = runWithVolumeKnob(t, true, "app", sourceDir, destDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range testMediaFiles {
		err := m.CheckExistsAt(m.FullExpectedDestination())
		if err != nil {
			t.Fatal(err)
		}

		err = m.CheckExistsAt(m.SourceDir)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func Test_ShouldCopy_WhenMediaDoesNotExist(t *testing.T) {
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

	err = runWithVolumeKnob(t, true, "app", sourceDir, destDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range testMediaFiles {
		err := m.CheckExistsAt(m.FullExpectedDestination())
		if err != nil {
			t.Fatal(err)
		}

		err = m.CheckExistsAt(m.SourceDir)
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
	var shouldBeMissing []TestMediaFile
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
			} else {
				shouldBeMissing = append(shouldBeMissing, *m)
			}
			jpgCount++
		case RafFile:
			if rafCount > 0 {
				m.CopyToExpectedDestination()
			} else {
				shouldBeMissing = append(shouldBeMissing, *m)
			}
			rafCount++
		case MovFile:
			if movCount > 0 {
				m.CopyToExpectedDestination()
			} else {
				shouldBeMissing = append(shouldBeMissing, *m)
			}
			movCount++
		default:
			panic("Type is not supported")
		}
	}

	err = runWithVolumeKnob(t, true, "app", "--move", sourceDir, destDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range testMediaFiles {
		err := m.CheckExistsAt(m.FullExpectedDestination())
		if err != nil {
			t.Fatal(err)
		}
	}

	for _, m := range shouldBeMissing {
		err = m.CheckMissingAt(m.SourceDir)
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

// should error or conflict when media with same name already exists but the hash doesn't match

func TestIntegration_ShouldError_WhenDestinationFolderDoesNotExist(t *testing.T) {
	sourceDir, err := os.MkdirTemp(".", "tmptest")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(sourceDir)

	err = runWithVolumeKnob(t, true, "app", sourceDir, "notExist")
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

	err = runWithVolumeKnob(t, true, "app", "notExist", destDir)
	if err == nil {
		t.Fatalf("expected an error, but got nil")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected error to be os.ErrNotExist, but got: %v", err)
	}
}

// TODO: test dynamic hash chunk using mocking
// test what happens when unsupported media is present

func runWithVolumeKnob(t *testing.T, silent bool, args ...string) error {
	if !silent {
		originalArgs := os.Args
		os.Args = args
		t.Cleanup(func() {
			os.Args = originalArgs
		})

		err := run()
		if err != nil {
			return err
		}
	} else {
		originalStdout := os.Stdout
		originalStderr := os.Stderr

		devNull, err := os.Open(os.DevNull)
		if err != nil {
			panic("failed to open /dev/null")
		}
		defer devNull.Close()

		os.Stdout = devNull
		os.Stderr = devNull

		defer func() {
			os.Stdout = originalStdout
			os.Stderr = originalStderr
		}()

		originalArgs := os.Args
		os.Args = args
		t.Cleanup(func() {
			os.Args = originalArgs
		})
		err = run()
		if err != nil {
			return err
		}
	}
	return nil
}

// TODO: add tests for move and copy
