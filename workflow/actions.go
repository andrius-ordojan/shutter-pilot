package workflow

import (
	"fmt"
	"io"
	"os"

	"github.com/andrius-ordojan/shutter-pilot/media"
)

type (
	actionType string
)

const (
	move     actionType = "move"
	copy     actionType = "copy"
	skip     actionType = "skip"
	conflict actionType = "conflict"
)

type action struct {
	execute func() (string, error)
	summery func() string
	aType   actionType
}

func newMoveAction(file media.File, destinationDir string) action {
	if file.GetPath() == "" {
		panic("path not set for media file")
	}
	if destinationDir == "" {
		panic("destination dir not set")
	}

	return action{
		aType: move,
		execute: func() (string, error) {
			dstPath, err := file.GetDestinationPath(destinationDir)
			if err != nil {
				return "", err
			}

			err = os.Rename(file.GetPath(), dstPath)
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("Moving from %s to %s", file.GetPath(), dstPath), nil
		},
		summery: func() string {
			return fmt.Sprintf("Move: %s", file.GetPath())
		},
	}
}

func newCopyAction(file media.File, destinationDir string) action {
	if file.GetPath() == "" {
		panic("path not set for media file")
	}
	if destinationDir == "" {
		panic("destination dir not set")
	}

	return action{
		aType: copy,
		execute: func() (string, error) {
			dstPath, err := file.GetDestinationPath(destinationDir)
			if err != nil {
				return "", err
			}

			sourceFile, err := os.Open(file.GetPath())
			if err != nil {
				return "", fmt.Errorf("failed to open source file: %w", err)
			}
			defer sourceFile.Close()

			destinationFile, err := os.Create(dstPath)
			if err != nil {
				return "", fmt.Errorf("failed to create destination file: %w", err)
			}
			defer destinationFile.Close()

			_, err = io.Copy(destinationFile, sourceFile)
			if err != nil {
				return "", fmt.Errorf("failed to copy content: %w", err)
			}

			err = destinationFile.Sync()
			if err != nil {
				return "", fmt.Errorf("failed to sync destination file: %w", err)
			}

			return fmt.Sprintf("Copying from %s to %s", file.GetPath(), dstPath), nil
		},
		summery: func() string {
			return fmt.Sprintf("Copy: %s", file.GetPath())
		},
	}
}

func newSkipAction(source, destination media.File) action {
	if source.GetPath() == "" {
		panic("path not set for source media file")
	}
	if destination.GetPath() == "" {
		panic("path not set for destination media file")
	}

	return action{
		aType: skip,
		execute: func() (string, error) {
			return fmt.Sprintf("Skipping %s", source.GetPath()), nil
		},
		summery: func() string {
			return fmt.Sprintf("Skip: %s (already exists at %s)", source.GetPath(), destination.GetPath())
		},
	}
}

func newConflictAction(conflictedFiles []media.File) action {
	if len(conflictedFiles) < 2 {
		panic("less than 2 files in conflicted files slice")
	}

	return action{
		aType: conflict,
		execute: func() (string, error) {
			return "conflict", nil
		},
		summery: func() string {
			firstConflict := conflictedFiles[0].GetPath()
			var restOfConflicts []string
			for _, f := range conflictedFiles[1:] {
				restOfConflicts = append(restOfConflicts, f.GetPath())
			}
			return fmt.Sprintf("Conflict: %s (has the same contents as %s)", firstConflict, restOfConflicts)
		},
	}
}
