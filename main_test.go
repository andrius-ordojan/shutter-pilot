package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	isValid             bool
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
	{Name: "DSCF9533.RAF", Type: RafFile, ExpectedDestination: "photos/2024/2024-12-07", isValid: true},
	{Name: "DSCF9533.JPG", Type: JpgFile, ExpectedDestination: "photos/2024/2024-12-07/sooc", isValid: true},
	{Name: "DSCF3517.JPG", Type: JpgFile, ExpectedDestination: "photos/2024/2024-11-13/sooc", isValid: true},
	{Name: "DSCF3517.RAF", Type: RafFile, ExpectedDestination: "photos/2024/2024-11-13", isValid: true},
	{Name: "DSCF9531.MOV", Type: MovFile, ExpectedDestination: "videos/2024/2024-12-07", isValid: true},
	{Name: "nometadata.MOV", Type: MovFile, isValid: false},
	{Name: "nometadata.JPG", Type: JpgFile, isValid: false},
}

func validTestMediaFiles() []*TestMediaFile {
	res := make([]*TestMediaFile, 0, len(testMediaFiles))
	for _, e := range testMediaFiles {
		if !e.isValid {
			continue
		}
		res = append(res, e)
	}
	return res
}

func invalidTestMediaFiles() []*TestMediaFile {
	res := make([]*TestMediaFile, 0, len(testMediaFiles))
	for _, e := range testMediaFiles {
		if e.isValid {
			continue
		}
		res = append(res, e)
	}
	return res
}

func makeSourceDirWithOutCleanup(t *testing.T) string {
	return makeSourceDir(t, false)
}

func makeSourceDirWithCleanup(t *testing.T) string {
	return makeSourceDir(t, true)
}

func makeSourceDir(t *testing.T, cleanup bool) string {
	sourceDir, err := os.MkdirTemp(".", "tmp_source")
	if err != nil {
		t.Error(err)
	}

	if cleanup {
		t.Cleanup(func() { os.RemoveAll(sourceDir) })
	}

	return sourceDir
}

func makeDestinationDirWithOutCleanup(t *testing.T) string {
	return makeDestinationDir(t, false)
}

func makeDestinationDirWithCleanup(t *testing.T) string {
	return makeDestinationDir(t, true)
}

func makeDestinationDir(t *testing.T, cleanup bool) string {
	destDir, err := os.MkdirTemp(".", "tmp_dest")
	if err != nil {
		t.Error(err)
	}

	if cleanup {
		t.Cleanup(func() { os.RemoveAll(destDir) })
	}

	return destDir
}

func runLoudly(t *testing.T, args ...string) error {
	return runWithVolume(t, false, args...)
}

func runSilently(t *testing.T, args ...string) error {
	return runWithVolume(t, true, args...)
}

func runWithVolume(t *testing.T, silent bool, args ...string) error {
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

func Test_ShouldSkip_WhenMediaExists(t *testing.T) {
	srcDir := makeSourceDirWithCleanup(t)
	destDir := makeDestinationDirWithCleanup(t)

	for _, m := range validTestMediaFiles() {
		m.SourceDir = srcDir
		m.DestinationDir = destDir
		m.CopyTo(srcDir)
		m.CopyToExpectedDestination()
	}

	err := runSilently(t, "app", srcDir, destDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range validTestMediaFiles() {
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
	srcDir := makeSourceDirWithCleanup(t)
	destDir := makeDestinationDirWithCleanup(t)

	jpgCount := 0
	rafCount := 0
	movCount := 0
	for _, m := range validTestMediaFiles() {
		m.SourceDir = srcDir
		m.DestinationDir = destDir
		m.CopyTo(srcDir)

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

	err := runSilently(t, "app", srcDir, destDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range validTestMediaFiles() {
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
	srcDir := makeSourceDirWithCleanup(t)
	destDir := makeDestinationDirWithCleanup(t)

	var shouldBeMissing []TestMediaFile
	jpgCount := 0
	rafCount := 0
	movCount := 0
	for _, m := range validTestMediaFiles() {
		m.SourceDir = srcDir
		m.DestinationDir = destDir
		m.CopyTo(srcDir)

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

	err := runSilently(t, "app", "--move", srcDir, destDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range validTestMediaFiles() {
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

func Test_ShouldProcessFiles_WhenTheyAreLocatedInSubfolders(t *testing.T) {
	srcDir := makeSourceDirWithCleanup(t)
	destDir := makeDestinationDirWithCleanup(t)

	for _, m := range validTestMediaFiles() {
		m.SourceDir = srcDir
		m.DestinationDir = destDir
		m.CopyTo(filepath.Join(srcDir, "subfolder"))
	}

	err := runSilently(t, "app", srcDir, destDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range validTestMediaFiles() {
		err := m.CheckExistsAt(m.FullExpectedDestination())
		if err != nil {
			t.Fatal(err)
		}
	}
}

func Test_ShouldSkip_WhenMediaContentIsSameButNameIsDifferent(t *testing.T) {
	srcDir := makeSourceDirWithCleanup(t)
	destDir := makeDestinationDirWithCleanup(t)

	media := testMediaFiles[0]
	media.SourceDir = srcDir
	media.DestinationDir = destDir
	media.CopyTo(srcDir)
	media.CopyToExpectedDestination()
	mediaCopy := *media

	newMediaName := "newname.raf"
	os.Rename(filepath.Join(media.FullExpectedDestination(), media.Name), filepath.Join(destDir, newMediaName))

	err := runSilently(t, "app", srcDir, destDir)
	if err != nil {
		t.Fatal(err)
	}

	err = media.CheckMissingAt(media.FullExpectedDestination())
	if err != nil {
		t.Fatal(err)
	}

	mediaCopy.Name = newMediaName
	err = mediaCopy.CheckExistsAt(mediaCopy.FullExpectedDestination())
	if err != nil {
		t.Fatal(err)
	}
}

func Test_ShouldMove_WhenMediaExistsButIsNotLocatedCorrectly(t *testing.T) {
	srcDir := makeSourceDirWithCleanup(t)
	destDir := makeDestinationDirWithCleanup(t)

	for _, m := range validTestMediaFiles() {
		m.SourceDir = srcDir
		m.DestinationDir = destDir
		m.CopyTo(destDir)
	}

	err := runSilently(t, "app", srcDir, destDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range validTestMediaFiles() {
		err := m.CheckExistsAt(m.FullExpectedDestination())
		if err != nil {
			t.Fatal(err)
		}
	}
}

func Test_ShouldConflict_WhenDuplicateMediaExistsInDestination(t *testing.T) {
	srcDir := makeSourceDirWithCleanup(t)
	destDir := makeDestinationDirWithCleanup(t)

	media := testMediaFiles[0]
	media.SourceDir = srcDir
	media.DestinationDir = destDir
	media.CopyTo(destDir)
	media.CopyTo(srcDir)
	media.CopyToExpectedDestination()

	newMediaName := "newname.raf"
	media.CopyTo(filepath.Join(destDir, "2024"))
	os.Rename(filepath.Join(destDir, "2024", media.Name), filepath.Join(destDir, "2024", newMediaName))

	err := runSilently(t, "app", srcDir, destDir)
	if err != nil {
		t.Fatal(err)
	}

	err = media.CheckExistsAt(media.FullExpectedDestination())
	if err != nil {
		t.Fatal(err)
	}
	err = media.CheckExistsAt(srcDir)
	if err != nil {
		t.Fatal(err)
	}
	err = media.CheckExistsAt(destDir)
	if err != nil {
		t.Fatal(err)
	}
	err = media.CheckExistsAt(media.FullExpectedDestination())
	if err != nil {
		t.Fatal(err)
	}
	mediaCopy := *media
	mediaCopy.Name = newMediaName
	err = mediaCopy.CheckExistsAt(filepath.Join(destDir, "2024"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegration_ShouldError_WhenDestinationFolderDoesNotExist(t *testing.T) {
	sourceDir, err := os.MkdirTemp(".", "tmptest")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(sourceDir)

	err = runSilently(t, "app", sourceDir, "notExist")
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

	err = runSilently(t, "app", "notExist", destDir)
	if err == nil {
		t.Fatalf("expected an error, but got nil")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected error to be os.ErrNotExist, but got: %v", err)
	}
}

func Test_ShouldNotMakeChanges_WhenDryrunIsOn(t *testing.T) {
	srcDir := makeSourceDirWithCleanup(t)
	destDir := makeDestinationDirWithCleanup(t)

	for _, m := range validTestMediaFiles() {
		m.SourceDir = srcDir
		m.DestinationDir = destDir
		m.CopyTo(srcDir)
	}

	err := runSilently(t, "app", "--dryrun", srcDir, destDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range validTestMediaFiles() {
		err := m.CheckMissingAt(m.FullExpectedDestination())
		if err != nil {
			t.Fatal(err)
		}

		err = m.CheckExistsAt(m.SourceDir)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func Test_ShouldError_WhenMetadataNotPresentInPhoto(t *testing.T) {
	srcDir := makeSourceDirWithCleanup(t)
	destDir := makeDestinationDirWithCleanup(t)

	for _, m := range invalidTestMediaFiles() {
		if m.Type != JpgFile {
			continue
		}
		m.SourceDir = srcDir
		m.DestinationDir = destDir
		m.CopyTo(srcDir)
	}

	err := runSilently(t, "app", srcDir, destDir)
	if err == nil {
		t.Fatal("execution shuold fail because exif data does not exist")
	}
}

func Test_ShouldError_WhenMetadataNotPresentInVideo(t *testing.T) {
	srcDir := makeSourceDirWithCleanup(t)
	destDir := makeDestinationDirWithCleanup(t)

	for _, m := range invalidTestMediaFiles() {
		if m.Type != MovFile {
			continue
		}
		m.SourceDir = srcDir
		m.DestinationDir = destDir
		m.CopyTo(srcDir)
	}

	err := runSilently(t, "app", srcDir, destDir)
	if err == nil {
		t.Fatal("execution shuold fail because metadata does not exist")
	}
}

func Test_ShouldCopyCertainFiletypes_WhenFilterIsSelected(t *testing.T) {
	srcDir := makeSourceDirWithCleanup(t)
	destDir := makeDestinationDirWithCleanup(t)

	for _, m := range validTestMediaFiles() {
		m.SourceDir = srcDir
		m.DestinationDir = destDir
		m.CopyTo(srcDir)
	}

	err := runSilently(t, "app", "-f", "jpg", srcDir, destDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range validTestMediaFiles() {
		if m.Type == JpgFile {
			err := m.CheckExistsAt(m.FullExpectedDestination())
			if err != nil {
				t.Fatal(err)
			}
		} else {
			err := m.CheckMissingAt(m.FullExpectedDestination())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func Test_ShouldNotUseSoocFolderForJpg_WhenNoSoocOptionIsSet(t *testing.T) {
	srcDir := makeSourceDirWithCleanup(t)
	destDir := makeDestinationDirWithCleanup(t)

	for _, m := range validTestMediaFiles() {
		m.SourceDir = srcDir
		m.DestinationDir = destDir
		m.CopyTo(srcDir)
	}

	err := runSilently(t, "app", "--nosooc", srcDir, destDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range validTestMediaFiles() {
		if m.Type == JpgFile {
			expectedPath := filepath.Dir(m.FullExpectedDestination())
			err := m.CheckExistsAt(expectedPath)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

// TODO:no subfolder setting test
// TODO: test multiple sources

func TestParseFileTypes(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      []string
		expectErr bool
	}{
		{
			name:      "Single file type",
			input:     "jpg",
			want:      []string{"jpg"},
			expectErr: false,
		},
		{
			name:      "Multiple file types without spaces",
			input:     "jpg,mov",
			want:      []string{"jpg", "mov"},
			expectErr: false,
		},
		{
			name:      "Multiple file types with spaces",
			input:     "jpg, mov, raf",
			want:      []string{"jpg", "mov", "raf"},
			expectErr: false,
		},
		{
			name:      "Trailing comma",
			input:     "jpg,mov,",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "Consecutive commas",
			input:     "jpg,,mov",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "Empty string",
			input:     "",
			want:      []string{},
			expectErr: true,
		},
		{
			name:      "Only commas",
			input:     ",,,",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "Spaces only",
			input:     "   ",
			want:      []string{},
			expectErr: true,
		},
		{
			name:      "Spaces around commas",
			input:     " jpg , mov ",
			want:      []string{"jpg", "mov"},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFileTypes(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("parseFileTypes() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !equalSlices(got, tt.want) {
				t.Errorf("parseFileTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidFileType(t *testing.T) {
	tests := []struct {
		name string
		ft   string
		want bool
	}{
		{"Valid lowercase jpg", "jpg", true},
		{"Valid lowercase mov", "mov", true},
		{"Valid lowercase raf", "raf", true},
		{"Valid uppercase JPG", "JPG", true},
		{"Valid mixed case Mov", "Mov", true},
		{"Invalid file type png", "png", false},
		{"Empty string", "", false},
		{"Whitespace", " ", false},
		{"Special characters", "jpg!", false},
		{"Numeric", "123", false},
		{"Partial match", "jp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidFileType(tt.ft)
			if got != tt.want {
				t.Errorf("isValidFileType(%q) = %v, want %v", tt.ft, got, tt.want)
			}
		})
	}
}

func TestValidateFileTypes(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		want      []string
		expectErr bool
	}{
		{
			name:      "No filter provided",
			filter:    "",
			want:      allowedFileTypes,
			expectErr: false,
		},
		{
			name:      "Valid single file type",
			filter:    "jpg",
			want:      []string{"jpg"},
			expectErr: false,
		},
		{
			name:      "Valid multiple file types",
			filter:    "jpg,mov",
			want:      []string{"jpg", "mov"},
			expectErr: false,
		},
		{
			name:      "Valid multiple file types with spaces",
			filter:    "jpg, mov, raf",
			want:      []string{"jpg", "mov", "raf"},
			expectErr: false,
		},
		{
			name:      "Invalid single file type",
			filter:    "png",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "Invalid multiple file types",
			filter:    "jpg,png",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "Trailing comma",
			filter:    "jpg,mov,",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "Consecutive commas",
			filter:    "jpg,,mov",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "Empty string after trimming",
			filter:    "   ",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "Mixed valid and invalid",
			filter:    "jpg,invalid,raf",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "Case insensitivity",
			filter:    "JPG,Mov",
			want:      []string{"JPG", "Mov"},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateFileTypes(tt.filter)
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateFileTypes() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !equalSlicesIgnoreCase(got, tt.want) {
				t.Errorf("ValidateFileTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func equalSlicesIgnoreCase(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if !strings.EqualFold(v, b[i]) {
			return false
		}
	}
	return true
}
