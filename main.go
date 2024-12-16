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

//
// func processFilesInDirectory(SourceDir string, destinationDir string, dryRun bool) {
// 	err := filepath.Walk(SourceDir, func(path string, info os.FileInfo, err error) error {
// 		if err != nil {
// 			return err
// 		}
//
// 		if info.IsDir() {
// 			return nil
// 		}
//
// 		ext := strings.ToLower(filepath.Ext(path))
// 		switch ext {
// 		case ".jpg":
// 			jpg := &Jpg{Path: path}
//
// 			exif, err := jpg.ReadExif()
// 			if err != nil {
// 				log.Fatal(err)
// 			}
// 			fmt.Println(exif)
//
// 			dateTime, err := exif.DateTime()
// 			if err != nil {
// 				log.Fatal(err)
// 			}
//
// 			mediaHome := createMediaDir(destinationDir, Photos, dateTime, "sooc", dryRun)
// 			destPath := filepath.Join(mediaHome, filepath.Base(path))
// 			movefile(path, destPath, dryRun)
// 		case ".raf":
// 			raf := &Raf{Path: path}
//
// 			exif, err := raf.ReadExif()
// 			if err != nil {
// 				log.Fatal(err)
// 			}
//
// 			dateTime, err := exif.DateTime()
// 			if err != nil {
// 				log.Fatal(err)
// 			}
//
// 			mediaHome := createMediaDir(destinationDir, Photos, dateTime, "", dryRun)
// 			destPath := filepath.Join(mediaHome, filepath.Base(path))
// 			movefile(path, destPath, dryRun)
// 		case ".mov":
// 			mov := &Mov{Path: path}
//
// 			creationTime, err := mov.ReadCreationTime()
// 			if err != nil {
// 				log.Fatal(err)
// 			}
//
// 			mediaHome := createMediaDir(destinationDir, Videos, creationTime, "", dryRun)
// 			destPath := filepath.Join(mediaHome, filepath.Base(path))
// 			movefile(path, destPath, dryRun)
// 		default:
// 			log.Fatalf("unsupported file: %s\n", path)
// 		}
//
// 		return nil
// 	})
// 	if err != nil {
// 		log.Fatalf("Error reading directory: %v", err)
// 	}
// }
//
// func createMediaDir(destinationDir string, mediaType MediaType, dateTime time.Time, subPath string, dryRun bool) string {
// 	date := dateTime.Format("2006-01-02")
// 	year := strconv.Itoa(dateTime.Year())
// 	path := filepath.Join(destinationDir, string(mediaType), year, date, subPath)
//
// 	if dryRun {
// 		fmt.Printf("creating directory: %s", path)
// 	} else {
// 		if _, err := os.Stat(path); os.IsNotExist(err) {
// 			err := os.MkdirAll(path, os.ModePerm)
// 			if err != nil {
// 				log.Fatalf("Error creating directory: %v", err)
// 			}
// 		}
// 	}
//
// 	return path
// }
//
// func movefile(source string, destination string, dryRun bool) {
// 	if dryRun {
// 		fmt.Printf("moving %s to %s", source, destination)
// 	} else {
// 		err := os.Rename(source, destination)
// 		if err != nil {
// 			log.Fatalf("error moving file: %v", err)
// 		}
// 	}
// }

// TODO: add io writer interface and make function for printing so I can choose where the output is going buf reader or os.stdout
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
			return err
		}
	}

	return nil

	// processFilesInDirectory(args.Source, args.Destination, args.DryRun)
	// BUG: _embeded jpg gets created next to raf files. don't do that
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
