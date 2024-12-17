package workflow

import (
	"fmt"

	"github.com/andrius-ordojan/shutter-pilot/media"
)

const (
	oneKB = 1024
	oneMB = 1024 * oneKB
	oneGB = 1024 * oneMB
)

type Plan struct {
	actions  []action
	moveMode bool
}

func (p *Plan) addAction(action action) {
	p.actions = append(p.actions, action)
}

func (p *Plan) Apply() error {
	fmt.Println("Applying plan:")

	for _, action := range p.actions {
		err := action.execute()
		if err != nil {
			return fmt.Errorf("error while executing action: %w", err)
		}
	}

	fmt.Printf("\n")
	return nil
}

func (p *Plan) printSummary() {
	moveCount := 0
	copyCount := 0
	skipCount := 0

	fmt.Println("Detailed Actions:")
	for _, action := range p.actions {
		switch action.aType {
		case move:
			fmt.Printf("  Move: %s\n", action.sourceMedia.GetPath())
			moveCount++
		case copy:
			fmt.Printf("  Copy: %s\n", action.sourceMedia.GetPath())
			copyCount++
		case skip:
			fmt.Printf("  Skip: %s (already exists at %s)\n", action.sourceMedia.GetPath(), action.destinationMedia.GetPath())
			skipCount++
		}
	}
	fmt.Printf("\n")

	fmt.Printf("Plan Summary:\n")
	fmt.Printf("  Files to move: %d\n", moveCount)
	fmt.Printf("  Files to copy: %d\n", copyCount)
	fmt.Printf("  Files skipped: %d\n", skipCount)

	fmt.Printf("\n")
}

func CreatePlan(sourcePath, destinationPath string, moveMode bool) (Plan, error) {
	sourceMedia, err := scanFiles(sourcePath)
	if err != nil {
		return Plan{}, fmt.Errorf("error occured while scanning source directory: %w", err)
	}
	sourceMap := make(map[string]media.File)
	for _, media := range sourceMedia {
		sourceMap[media.GetFingerprint()] = media
	}

	destinationMedia, err := scanFiles(destinationPath)
	if err != nil {
		return Plan{}, fmt.Errorf("error occured while scanning destination directory: %w", err)
	}
	destMap := make(map[string]media.File)
	for _, media := range destinationMedia {
		destMap[media.GetFingerprint()] = media
	}

	plan := Plan{moveMode: moveMode}

	// TODO: handle when media exists but is not orginized correctly so need to implement check for correct placement of destination media
	// media should determine it's own destinationPath then I can check correctness with current path
	// need to create a new loop for destiniation map and check if everything is placed correctly

	for _, destMedia := range destMap {
		mediaDestPath, err := destMedia.GetDestinationPath(destinationPath)
		if err != nil {
			return Plan{}, err
		}

		if destMedia.GetPath() != mediaDestPath {
			plan.addAction(action{
				aType:          move,
				sourceMedia:    destMedia,
				destinationDir: destinationPath,
			})
		}

		// TODO: create error for duplicate media
	}

	for hash, srcMedia := range sourceMap {
		if destMedia, exists := destMap[hash]; exists {
			plan.addAction(action{
				aType:            skip,
				sourceMedia:      srcMedia,
				destinationMedia: destMedia,
				destinationDir:   destinationPath,
			})
		} else {
			if moveMode {
				plan.addAction(action{
					aType:          move,
					sourceMedia:    srcMedia,
					destinationDir: destinationPath,
				})
			} else {
				plan.addAction(action{
					aType:          copy,
					sourceMedia:    srcMedia,
					destinationDir: destinationPath,
				})
			}
		}
	}

	plan.printSummary()

	return plan, nil
}