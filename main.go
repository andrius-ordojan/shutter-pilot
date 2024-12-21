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

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
