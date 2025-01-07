package workflow

import (
	"context"
	"errors"
	"fmt"
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
			return err
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
	var skippedSummeries string
	var copySummeries string
	var moveSummeries string
	var conflictSummeries string

	fmt.Println("Detailed Actions:")
	for _, action := range p.actions {
		summery := action.summery()

		switch action.aType {
		case move:
			moveSummeries += fmt.Sprintf("  %s\n", summery)
			moveCount++
		case copy:
			copySummeries += fmt.Sprintf("  %s\n", summery)
			copyCount++
		case skip:
			skippedSummeries += fmt.Sprintf("  %s\n", summery)
			skipCount++
		case conflict:
			conflictSummeries += fmt.Sprintf("  %s\n", summery)
			conflictCount++
		}
	}
	fmt.Print(skippedSummeries)
	fmt.Print(copySummeries)
	fmt.Print(moveSummeries)
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

func CreatePlan(ctx context.Context, sourcePaths []string, destinationPath string, moveMode bool, filter []string, noSooc bool) (Plan, error) {
	fmt.Println("building execution plan... (depending on disk used and number of files this might take a while)")

	mediaMaps, err := prepareMediaMaps(ctx, sourcePaths, destinationPath, filter, noSooc)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return Plan{}, errors.New("Plan creation canceled")
		}
		return Plan{}, err
	}

	plan := Plan{moveMode: moveMode}

	for _, files := range mediaMaps.DestMap {
		if len(files) > 1 {
			plan.addAction(newConflictAction(files))
		}
	}

	for _, e := range mediaMaps.DestMap {
		mediaDestPath, err := e[0].GetDestinationPath(destinationPath)
		if err != nil {
			return Plan{}, fmt.Errorf("%s %w", e[0].GetPath(), err)
		}

		if e[0].GetPath() != mediaDestPath {
			plan.addAction(newMoveAction(e[0], destinationPath))
		}
	}

	for hash, srcMedia := range mediaMaps.SourceMap {
		if e, exists := mediaMaps.DestMap[hash]; exists {
			plan.addAction(newSkipAction(srcMedia, e[0]))
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
