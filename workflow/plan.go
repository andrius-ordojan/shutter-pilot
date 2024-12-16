package workflow

import (
	"fmt"

	"github.com/andrius-ordojan/shutter-pilot/media"
)

const (
	OneKB = 1024
	OneMB = 1024 * OneKB
	OneGB = 1024 * OneMB
)

type Plan struct {
	Actions  []Action
	MoveMode bool
}

func (p *Plan) AddAction(action Action) {
	p.Actions = append(p.Actions, action)
}

func (p *Plan) Apply() error {
	fmt.Println("Applying plan:")

	for _, action := range p.Actions {
		err := action.Execute()
		if err != nil {
			return fmt.Errorf("error while executing action: %w", err)
		}
	}

	fmt.Printf("\n")
	return nil
}

func (p *Plan) PrintSummary() {
	moveCount := 0
	copyCount := 0
	skipCount := 0

	fmt.Println("Detailed Actions:")
	for _, action := range p.Actions {
		switch action.Type {
		case ActionMove:
			fmt.Printf("  Move: %s\n", action.SourceMedia.GetPath())
			moveCount++
		case ActionCopy:
			fmt.Printf("  Copy: %s\n", action.SourceMedia.GetPath())
			copyCount++
		case ActionSkip:
			fmt.Printf("  Skip: %s (already exists at %s)\n", action.SourceMedia.GetPath(), action.DestinationMedia.GetPath())
			skipCount++
		}
	}
	fmt.Printf("\n")

	fmt.Printf("Plan Summary:\n")
	if p.MoveMode {
		fmt.Printf("  Files to move: %d\n", moveCount)
	} else {
		fmt.Printf("  Files to copy: %d\n", copyCount)
	}
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

	plan := Plan{MoveMode: moveMode}

	// TODO: handle when media exists but is not orginized correctly so need to implement check for correct placement of destination media
	// media should determine it's own destinationPath then I can check correctness with current path
	// need to create a new loop for destiniation map and check if everything is placed correctly

	// for hash, destMedia := range destMap {
	// 	// check if path is correct
	//
	// 	// create error for duplicate media
	// }

	for hash, srcMedia := range sourceMap {
		if destMedia, exists := destMap[hash]; exists {
			plan.AddAction(Action{
				Type:             ActionSkip,
				SourceMedia:      srcMedia,
				DestinationMedia: destMedia,
				DestinationDir:   destinationPath,
			})
		} else {
			if moveMode {
				plan.AddAction(Action{
					Type:           ActionMove,
					SourceMedia:    srcMedia,
					DestinationDir: destinationPath,
				})
			} else {
				plan.AddAction(Action{
					Type:           ActionCopy,
					SourceMedia:    srcMedia,
					DestinationDir: destinationPath,
				})
			}
		}
	}

	plan.PrintSummary()

	return plan, nil
}
