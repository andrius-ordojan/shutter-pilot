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

	for _, a := range p.actions {
		if a.aType == conflict {
			fmt.Println("  File conflicts need to be resolved before application can proceed. Resolve them and rerun application to continue.")
			return nil
		}
	}

	for _, action := range p.actions {
		result, err := action.execute()
		if err != nil {
			return nil
		}

		fmt.Printf("  %s\n", result)
	}

	fmt.Printf("\n")
	return nil
}

func (p *Plan) printSummary() error {
	moveCount := 0
	copyCount := 0
	skipCount := 0
	conflictCount := 0

	fmt.Println("Detailed Actions:")
	var actionSummeries string
	var conflictSummeries string
	for _, action := range p.actions {
		summery := action.summery()

		if action.aType == conflict {
			conflictSummeries += fmt.Sprintf("  %s\n", summery)
		} else {
			actionSummeries += fmt.Sprintf("  %s\n", summery)
		}

		switch action.aType {
		case move:
			moveCount++
		case copy:
			copyCount++
		case skip:
			skipCount++
		case conflict:
			conflictCount++
		}
	}
	fmt.Print(actionSummeries)
	fmt.Print(conflictSummeries)

	fmt.Printf("\n")
	fmt.Printf("Plan Summary:\n")
	fmt.Printf("  Files to move: %d\n", moveCount)
	fmt.Printf("  Files to copy: %d\n", copyCount)
	fmt.Printf("  Files skipped: %d\n", skipCount)
	if conflictCount > 0 {
		fmt.Printf("  Detected conflicts: %d (will prevent execution of plan and reported actions might be incorrect)\n", conflictCount)
	} else {
		fmt.Printf("  Detected conflicts: %d\n", conflictCount)
	}
	fmt.Printf("\n")

	return nil
}

func CreatePlan(sourcePath, destinationPath string, moveMode bool) (Plan, error) {
	fmt.Println("building execution plan... (depending on disk used and number of files this might take a while)")

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
	destMap := make(map[string][]media.File)
	for _, media := range destinationMedia {
		destMap[media.GetFingerprint()] = append(destMap[media.GetFingerprint()], media)
	}

	plan := Plan{moveMode: moveMode}

	for _, files := range destMap {
		if len(files) > 1 {
			plan.addAction(newConflictAction(files))
		}
	}

	for _, e := range destMap {
		destMedia := e[0]

		mediaDestPath, err := destMedia.GetDestinationPath(destinationPath)
		if err != nil {
			return Plan{}, err
		}

		if destMedia.GetPath() != mediaDestPath {
			plan.addAction(newMoveAction(destMedia, destinationPath))
		}
	}

	for hash, srcMedia := range sourceMap {
		if e, exists := destMap[hash]; exists {
			destMedia := e[0]
			plan.addAction(newSkipAction(srcMedia, destMedia))
		} else {
			if moveMode {
				plan.addAction(newMoveAction(srcMedia, destinationPath))
			} else {
				plan.addAction(newCopyAction(srcMedia, destinationPath))
			}
		}
	}

	err = plan.printSummary()
	if err != nil {
		return Plan{}, fmt.Errorf("error occured while printing plan summery: %w", err)
	}

	return plan, nil
}
