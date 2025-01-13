package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/andrius-ordojan/shutter-pilot/workflow"
)

var allowedFileTypes = []string{"jpg", "raf", "mov"}

type args struct {
	Sources     string `arg:"positional,required" help:"source directories for media. Provide as a comma-separated list, e.g., /path/1,/path2/"`
	Destination string `arg:"positional,required" help:"destination directory for orginised media"`
	Filter      string `arg:"-f,--filter" help:"Filter by file types (allowed: jpg, raf, mov). Provide as a comma-separated list, e.g., -f jpg,mov"`
	MoveMode    bool   `arg:"-m,--move" default:"false" help:"moves files instead of copying"`
	DryRun      bool   `arg:"-d,--dryrun" default:"false" help:"does not modify file system"`
	NoSooc      bool   `arg:"-s,--nosooc" default:"false" help:"Does no place jpg photos under sooc directory, but next to raw files"`
}

func (args) Description() string {
	return "Compares media files in source directories with destination directory and organises them"
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

func validateFileTypes(filter string) ([]string, error) {
	if filter == "" {
		return allowedFileTypes, nil
	}

	parsedFileTypes, err := parseCommaSeperatedArg(filter)
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

func parseCommaSeperatedArg(arg string) ([]string, error) {
	parts := strings.Split(arg, ",")

	var args []string
	for i, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			return nil, fmt.Errorf("empty argument detected at position %d", i+1)
		}
		args = append(args, trimmed)
	}

	return args, nil
}

func validateSources(sources string) ([]string, error) {
	if sources == "" {
		return nil, errors.New("sources cannot be empty")
	}

	return parseCommaSeperatedArg(sources)
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	var args args
	parser := arg.MustParse(&args)

	filterByFiletypes, err := validateFileTypes(args.Filter)
	if err != nil {
		parser.Fail(err.Error())
	}

	sourcesList, err := validateSources(args.Sources)
	if err != nil {
		parser.Fail(err.Error())
	}

	plan, err := workflow.CreatePlan(ctx, sourcesList, args.Destination, args.MoveMode, filterByFiletypes, args.NoSooc)
	if err != nil {
		return err
	}

	if !args.DryRun {
		err := plan.Apply(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return errors.New("application shutting down gracefully")
			}

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
