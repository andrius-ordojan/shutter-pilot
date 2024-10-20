package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const appleEpochAdjustment = 2082844800

const (
	movieResourceAtomType   = "moov"
	movieHeaderAtomType     = "mvhd"
	referenceMovieAtomType  = "rmra"
	compressedMovieAtomType = "cmov"
)

func main() {
	sourcedir := flag.String("source", "", "The folder containing all the non-organized videos. Folders will be ignored.")
	destdir := flag.String("dest", "", "The folder where all videos will be moved to and organized")

	flag.Parse()

	if *sourcedir == "" {
		log.Fatal("Error: sourcedir is required")
	}
	if *destdir == "" {
		log.Fatal("Error: destdir is required")
	}

	if _, err := os.Stat(*sourcedir); os.IsNotExist(err) {
		log.Fatalf("source directory does not exist: %s", *sourcedir)
	}

	if _, err := os.Stat(*destdir); os.IsNotExist(err) {
		log.Fatalf("dest directory does not exist: %s", *destdir)
	}

	files, err := os.ReadDir(*sourcedir)
	if err != nil {
		log.Fatalf("Error reading directory: %v", err)
	}

	videoStructure := make(map[int][]string)
	for _, file := range files {
		if !file.IsDir() {
			fullPath := filepath.Join(*sourcedir, file.Name())
			fmt.Printf("Processing file: %s\n", fullPath)

			file, err := os.Open(fullPath)
			if err != nil {
				fmt.Println("Error opening file:", err)
				return
			}

			created, err := getVideoCreationTimeMetadata(file)
			file.Close()
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			year := created.Year()
			// strconv.Itoa(created.Year())
			videoStructure[year] = append(videoStructure[year], fullPath)
		}
	}

	fmt.Println(videoStructure)

	return
	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	created, err := getVideoCreationTimeMetadata(file)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Printf("Movie created at (%s)\n", strconv.Itoa(created.Year()))
}

func getVideoCreationTimeMetadata(videoBuffer io.ReadSeeker) (time.Time, error) {
	buf := make([]byte, 8)

	// Traverse videoBuffer to find movieResourceAtom
	for {
		// bytes 1-4 is atom size, 5-8 is type
		// Read atom
		if _, err := videoBuffer.Read(buf); err != nil {
			return time.Time{}, err
		}

		if bytes.Equal(buf[4:8], []byte(movieResourceAtomType)) {
			break // found it!
		}

		atomSize := binary.BigEndian.Uint32(buf) // check size of atom
		videoBuffer.Seek(int64(atomSize)-8, 1)   // jump over data and set seeker at beginning of next atom
	}

	// read next atom
	if _, err := videoBuffer.Read(buf); err != nil {
		return time.Time{}, err
	}

	atomType := string(buf[4:8]) // skip size and read type
	switch atomType {
	case movieHeaderAtomType:
		// read next atom
		if _, err := videoBuffer.Read(buf); err != nil {
			return time.Time{}, err
		}

		// byte 1 is version, byte 2-4 is flags, 5-8 Creation time
		appleEpoch := int64(binary.BigEndian.Uint32(buf[4:])) // Read creation time

		return time.Unix(appleEpoch-appleEpochAdjustment, 0).Local(), nil
	case compressedMovieAtomType:
		return time.Time{}, errors.New("compressed video")
	case referenceMovieAtomType:
		return time.Time{}, errors.New("reference video")
	default:
		return time.Time{}, errors.New("did not find movie header atom (mvhd)")
	}
}
