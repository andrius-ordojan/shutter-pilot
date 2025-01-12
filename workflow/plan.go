package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	oneKB = 1024
	oneMB = 1024 * oneKB
	oneGB = 1024 * oneMB
)

type Plan struct {
	actions []action
}

func (p *Plan) addAction(action action) {
	p.actions = append(p.actions, action)
}

func (p *Plan) handleDestinationsConflicts(mediaMaps *MediaMaps) {
	for _, files := range mediaMaps.DestMap {
		if len(files) > 1 {
			p.addAction(newConflictAction(files))
		}
	}
}

func (p *Plan) handleDestinationFiles(mediaMaps *MediaMaps, destinationPath string) error {
	for _, e := range mediaMaps.DestMap {
		mediaDestPath, err := e[0].GetDestinationPath(destinationPath)
		if err != nil {
			return fmt.Errorf("%s %w", e[0].GetPath(), err)
		}

		if e[0].GetPath() != mediaDestPath {
			p.addAction(newMoveAction(e[0], destinationPath))
		}
	}

	return nil
}

func (p *Plan) handleSourceFiles(mediaMaps *MediaMaps, moveMode bool, destinationPath string) {
	for hash, srcMedia := range mediaMaps.SourceMap {
		if e, exists := mediaMaps.DestMap[hash]; exists {
			p.addAction(newSkipAction(srcMedia, e[0]))
		} else {
			if moveMode {
				p.addAction(newMoveAction(srcMedia, destinationPath))
			} else {
				p.addAction(newCopyAction(srcMedia, destinationPath))
			}
		}
	}
}

func (p *Plan) Apply(ctx context.Context) error {
	fmt.Println("Applying plan:")
	var builder strings.Builder

	for _, a := range p.actions {
		if a.aType == conflict {
			fmt.Println("  File conflicts need to be resolved before application can proceed. Resolve them and rerun application to continue.")
			return nil
		}
	}

	for _, action := range p.actions {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			result, err := action.execute()
			if err != nil {
				return err
			}

			builder.WriteString(fmt.Sprintf("  %s\n", result))
		}
	}
	fmt.Print(builder.String())

	return nil
}

func (p *Plan) printSummary() error {
	moveCount := 0
	copyCount := 0
	skipCount := 0
	conflictCount := 0
	var skippedSummeries strings.Builder
	var copySummeries strings.Builder
	var moveSummeries strings.Builder
	var conflictSummeries strings.Builder

	fmt.Println("Detailed Actions:")
	for _, action := range p.actions {
		summery := action.summery()

		switch action.aType {
		case move:
			moveSummeries.WriteString(fmt.Sprintf("  %s\n", summery))
			moveCount++
		case copy:
			copySummeries.WriteString(fmt.Sprintf("  %s\n", summery))
			copyCount++
		case skip:
			skippedSummeries.WriteString(fmt.Sprintf("  %s\n", summery))
			skipCount++
		case conflict:
			conflictSummeries.WriteString(fmt.Sprintf("  %s\n", summery))
			conflictCount++
		}
	}
	fmt.Print(skippedSummeries.String())
	fmt.Print(copySummeries.String())
	fmt.Print(moveSummeries.String())
	fmt.Print(conflictSummeries.String())

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
	fmt.Println()

	mediaMaps, err := prepareMediaMaps(ctx, sourcePaths, destinationPath, filter, noSooc)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return Plan{}, errors.New("Plan creation interrupted")
		}
		return Plan{}, err
	}

	plan := Plan{}

	plan.handleDestinationsConflicts(&mediaMaps)
	err = plan.handleDestinationFiles(&mediaMaps, destinationPath)
	if err != nil {
		return Plan{}, err
	}
	plan.handleSourceFiles(&mediaMaps, moveMode, destinationPath)

	err = plan.printSummary()
	if err != nil {
		return Plan{}, fmt.Errorf("error occured while printing plan summery: %w", err)
	}

	return plan, nil
}
