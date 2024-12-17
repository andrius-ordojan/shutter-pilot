package workflow

import (
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
	move actionType = "move"
	copy actionType = "copy"
	skip actionType = "skip"
)

type action struct {
	aType actionType
	// TODO: change to from and to maybe?
	sourceMedia      media.File
	destinationMedia media.File
	destinationDir   string
}

func (a *action) execute() error {
	switch a.aType {
	case copy:
		dstPath, err := a.sourceMedia.GetDestinationPath(a.destinationDir)
		if err != nil {
			return nil
		}
		fmt.Printf("  Copying from %s to %s\n", a.sourceMedia.GetPath(), dstPath)

		sourceFile, err := os.Open(a.sourceMedia.GetPath())
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
	case move:
		dstPath, err := a.sourceMedia.GetDestinationPath(a.destinationDir)
		if err != nil {
			return nil
		}

		fmt.Printf("  Moving from %s to %s\n", a.sourceMedia.GetPath(), dstPath)

		err = os.Rename(a.sourceMedia.GetPath(), dstPath)
		if err != nil {
			log.Fatalf("error moving file: %v", err)
		}
	case skip:
		fmt.Printf("  Skipping %s\n", a.sourceMedia.GetPath())
	default:
		panic(fmt.Errorf("unknown action type: %s", a.aType))
	}
	return nil
}
