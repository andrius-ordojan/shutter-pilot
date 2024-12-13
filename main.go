package main

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/andrius-ordojan/shutter-pilot/workflow"
)

type args struct {
	Source      string `arg:"positional,required" help:"source directory for media"`
	Destination string `arg:"positional,required" help:"destination directory for orginised media"`
	MoveMode    bool   `arg:"-m,--move" default:"false" help:"moves files instead of copying"`
	DryRun      bool   `arg:"-d,--dryrun" default:"false" help:"does not modify file system"`
}

func (args) Description() string {
	return "Orginizes photo and video media into lightroom style directory structure"
}

func run() error {
	var args args
	arg.MustParse(&args)

	// BUG: creates directories if in dryrun mode
	plan, err := workflow.CreatePlan(args.Source, args.Destination, args.MoveMode)
	if err != nil {
		return err
	}

	if !args.DryRun {
		err := plan.Apply()
		if err != nil {
			return fmt.Errorf("error while executing action: %w", err)
		}
	}

	// TODO: print execution time to measure performance
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// TODO: create plan before making changes making sure there are no conflicts and create a report with changes this will be either --plan or --dry-run
// TODO: never overwrite existing files
// TODO: clean up source dir if it's empty of content
