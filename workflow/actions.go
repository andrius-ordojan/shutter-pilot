package workflow

import (
	"errors"
	"fmt"
	"io"
	"log"
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
	summery func() (string, error)
	aType   actionType
}

func newMoveAction(file media.File, destinationDir string) action {
	return action{
		aType: move,
		execute: func() (string, error) {
			if file.GetPath() == "" {
				return "", errors.New("media file path not specified")
			}

			dstPath, err := file.GetDestinationPath(destinationDir)
			if err != nil {
				return "", nil
			}

			err = os.Rename(file.GetPath(), dstPath)
			if err != nil {
				log.Fatalf("error moving file: %v", err)
			}

			return fmt.Sprintf("Moving from %s to %s", file.GetPath(), dstPath), nil
		},
		summery: func() (string, error) {
			if file.GetPath() == "" {
				return "", errors.New("media file path not specified")
			}
			return fmt.Sprintf("Move: %s", file.GetPath()), nil
		},
	}
}

func newCopyAction(file media.File, destinationDir string) action {
	return action{
		aType: copy,
		execute: func() (string, error) {
			if file.GetPath() == "" {
				return "", errors.New("media file path not specified")
			}
			if destinationDir == "" {
				return "", errors.New("destination directory not specified")
			}

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
		summery: func() (string, error) {
			if file.GetPath() == "" {
				return "", errors.New("media file path not specified")
			}

			return fmt.Sprintf("Copy: %s", file.GetPath()), nil
		},
	}
}

func newSkipAction(source, destination media.File) action {
	return action{
		aType: skip,
		execute: func() (string, error) {
			if source.GetPath() == "" {
				return "", errors.New("source media path not specified")
			}
			if source.GetPath() == "" {
				return "", errors.New("destination media path not specified")
			}

			return fmt.Sprintf("Skipping %s", source.GetPath()), nil
		},
		summery: func() (string, error) {
			if source.GetPath() == "" {
				return "", errors.New("source media path not specified")
			}
			if destination.GetPath() == "" {
				return "", errors.New("destination media path not specified")
			}

			return fmt.Sprintf("Skip: %s (already exists at %s)", source.GetPath(), destination.GetPath()), nil
		},
	}
}

func newConflictAction() action {
	return action{
		aType: conflict,
		execute: func() (string, error) {
			return "conflict", nil
		},
		summery: func() (string, error) {
			return "", nil
		},
	}
}
