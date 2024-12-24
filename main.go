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
	Filter      string `arg:"-f,--filter" help:"Filter by file types (allowed: jpg, raf, mov). Provide as a comma-separated list, e.g., -f jpg,mov"`
	MoveMode    bool   `arg:"-m,--move" default:"false" help:"moves files instead of copying"`
	DryRun      bool   `arg:"-d,--dryrun" default:"false" help:"does not modify file system"`
	// TODO: add option to disable jpg sooc subpath
}

func (args) Description() string {
	return "Orginizes photo and video media into lightroom style directory structure"
}

func parseFileTypes(input string) ([]string, error) {
	parts := strings.Split(input, ",")

	fileTypes := make([]string, 0, len(allowedFileTypes))
	for i, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			return nil, fmt.Errorf("empty file type detected at position %d", i+1)
		}
		fileTypes = append(fileTypes, trimmed)
	}

	return fileTypes, nil
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

func ValidateFileTypes(filter string) ([]string, error) {
	if filter == "" {
		return allowedFileTypes, nil
	}

	parsedFileTypes, err := parseFileTypes(filter)
	if err != nil {
		return nil, err
	}

	for _, ft := range parsedFileTypes {
		if !isValidFileType(ft) {
			return nil, fmt.Errorf("invalid file type: %s. Allowed types are: %s", ft, strings.Join(allowedFileTypes, ", "))
		}
	}

	return parsedFileTypes, nil
}

func run() error {
	var args args
	parser := arg.MustParse(&args)

	filterByFiletypes, err := ValidateFileTypes(args.Filter)
	if err != nil {
		parser.Fail(err.Error())
	}

	plan, err := workflow.CreatePlan(args.Source, args.Destination, args.MoveMode, filterByFiletypes)
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
