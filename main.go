package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/andrius-ordojan/shutter-pilot/workflow"
)

var allowedFileTypes = []string{"jpg", "raf", "mov"}

// TODO: change cli so I can have multiple sources and the last param will be destination ex: app source source dest
// BUG: parameter positions are broken. = is required not sure if bug
type args struct {
	Source      string `arg:"positional,required" help:"source directory for media"`
	Destination string `arg:"positional,required" help:"destination directory for orginised media"`
	FileTypes   string `arg:"-f,--filter" help:"Filter by file types (allowed: jpg, raf, mov). Can be specified multiple times."`
	MoveMode    bool   `arg:"-m,--move" default:"false" help:"moves files instead of copying"`
	DryRun      bool   `arg:"-d,--dryrun" default:"false" help:"does not modify file system"`
}

func (args) Description() string {
	return "Orginizes photo and video media into lightroom style directory structure"
}

func run() error {
	var args args
	arg.MustParse(&args)

	fmt.Println(args.FileTypes)
	// for _, ft := range args.FileTypes {
	// 	if !isValidFileType(ft) {
	// 		parser.Fail(fmt.Sprintf("Invalid file type: %s. Allowed types are: %s", ft, strings.Join(allowedFileTypes, ", ")))
	// 	}
	// }
	// fmt.Printf("Filtering by file types: %s\n", strings.Join(args.FileTypes, ", "))
	os.Exit(1)
	plan, err := workflow.CreatePlan(args.Source, args.Destination, args.MoveMode)
	if err != nil {
		return err
	}

	if !args.DryRun {
		err := plan.Apply()
		if err != nil {
			return fmt.Errorf("error while applying plan: %w", err)
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

func isValidFileType(ft string) bool {
	ft = strings.ToLower(ft)
	for _, allowed := range allowedFileTypes {
		if ft == allowed {
			return true
		}
	}
	return false
}
