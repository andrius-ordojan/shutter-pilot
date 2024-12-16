package workflow

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/andrius-ordojan/shutter-pilot/media"
)

type (
	ActionType string
)

const (
	ActionMove ActionType = "move"
	ActionCopy ActionType = "copy"
	ActionSkip ActionType = "skip"
)

type Action struct {
	Type             ActionType
	SourceMedia      media.File
	DestinationMedia media.File
	DestinationDir   string
}

func (a *Action) Execute() error {
	switch a.Type {
	case ActionCopy:
		dstPath, err := a.SourceMedia.GetDestinationPath(a.DestinationDir)
		if err != nil {
			return nil
		}
		fmt.Printf("  Copying from %s to %s\n", a.SourceMedia.GetPath(), dstPath)

		sourceFile, err := os.Open(a.SourceMedia.GetPath())
		if err != nil {
			return fmt.Errorf("failed to open source file: %w", err)
		}
		defer sourceFile.Close()

		destinationFile, err := os.Create(dstPath)
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
	case ActionMove:
		dstPath, err := a.SourceMedia.GetDestinationPath(a.DestinationDir)
		if err != nil {
			return nil
		}

		fmt.Printf("  Moving from %s to %s\n", a.SourceMedia.GetPath(), dstPath)

		err = os.Rename(a.SourceMedia.GetPath(), dstPath)
		if err != nil {
			log.Fatalf("error moving file: %v", err)
		}
	case ActionSkip:
		fmt.Printf("  Skipping %s\n", a.SourceMedia.GetPath())
	default:
		panic(fmt.Errorf("unknown action type: %s", a.Type))
	}
	return nil
}
