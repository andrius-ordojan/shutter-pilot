# shutter-pilot

Shutter-Pilot is a lightweight, stateless CLI tool that helps photographers manage and organize their media files with confidence. By comparing source and destination directories using content-based hashing, it ensures no files are missed or duplicated. It also organizes media into a date-based folder structure, inspired by Lightroom.

The resulting file structure looks like this

```
.
├── photos
│   ├── 2024
│   │   └── 2024-12-28
│   │       ├── DSCF7001.RAF
│   │       ├── DSCF7002.RAF
│   │       ├── DSCF7003.RAF
│   │       ├── DSCF7004.RAF
│   │       ├── DSCF7005.RAF
│   │       └── sooc
│   │           ├── DSCF7001.JPG
│   │           ├── DSCF7002.JPG
│   │           ├── DSCF7003.JPG
│   │           ├── DSCF7004.JPG
│   │           └── DSCF7005.JPG
│   └── 2025
│       └── 2025-01-12
│           ├── DSCF0001.RAF
│           ├── DSCF0002.RAF
│           └── sooc
│               ├── DSCF0001.JPG
│               └── DSCF0002.JPG
└── videos
    └── 2024
        └── 2024-12-25
            └── DSCF6246.MOV
```

---

## Features

- **Stateless**  
  Every run is independent no database or persistent state is required.

- **Content-Based Comparison**  
  Compares files by hashing their content, ensuring accuracy regardless of filename.

- **Organized File Structure**  
  Automatically organizes media into an easy-to-browse, date-based directory structure inspired by Lightroom.

- **Multiple Media Formats Supported**  
  Works seamlessly with JPG, RAF, and MOV files, with metadata extraction tailored for each format.

- **Recursive Directory Scanning**  
  Reads all files in a directory and its subdirectories.

- **Dry Run Mode**  
  Preview changes without modifying the file system.

- **Conflict Detection**  
  Identifies duplicate files based on their hashes and flags conflicts for manual resolution.

- **Flexible Input Handling**  
  Supports multiple source directories and allows filtering by file types (e.g., JPG, RAF, MOV).

- **Customizable File Placement**  
  Provides options to exclude or include "sooc" subfolders for JPG files.

## installation

### download binary

Go to the releases page and download the binary for your system.

Place the binary in your PATH.

### build from source

Use go install to build and install the binary

```bash
go install github.com/andrius-ordojan/shutter-pilot@main
```

This will install the binary in your go bin directory. Make sure it's in your PATH.

## usage

```
Compares media files in source directories with destination directory and organises them
Usage: shutter-pilot [--filter FILTER] [--move] [--dryrun] [--nosooc] SOURCES DESTINATION

Positional arguments:
  SOURCES                source directories for media. Provide as a comma-separated list, e.g., /path/1,/path2/
  DESTINATION            destination directory for orginised media

Options:
  --filter FILTER, -f FILTER
                         Filter by file types (allowed: jpg, raf, mov). Provide as a comma-separated list, e.g., -f jpg,mov
  --move, -m             moves files instead of copying [default: false]
  --dryrun, -d           does not modify file system [default: false]
  --nosooc, -s           Does no place jpg photos under sooc directory, but next to raw files [default: false]
  --help, -h             display this help and exit
```

## examples

Compare source and destination directories. Copy the missing files to the destination directory and organize them.

```bash
shutter-pilot /path/to/source /path/to/destination
```

Use multiple source directories

```bash
shutter-pilot /path/to/source1,/path/to/source2 /path/to/destination
```

Do not apply changes to the file system. Only show what would be done.

```bash
shutter-pilot --dryrun /path/to/source /path/to/destination
```

Only handle jpg files and don't use the dedicated sooc subfolder. This will place jpg files under the date folder.

```bash
shutter-pilot --filter jpg --nosooc /path/to/source /path/to/destination
```

Filter multiple file types

```bash
shutter-pilot --filter jpg,raf /path/to/source /path/to/destination
```

## file conflicts

If the destination folder contains duplicate files. Meaning they generate the exact same hash. It will result in a conflict. Duplicates can be addressed in multiple ways. For example renaming the file, skipping the file. Instead of making these decisions for you, the tool will list the conflicts and will not proceed to execute. You have to address these conflicts manually and rerun the tool.

## how it works

File comparison is based on hashing file content (fingerprinting). To optimize performance for large files (e.g., MOV), the tool hashes only the first and last segments of the file (1–10 MB based on file size). Sorting relies on EXIF metadata for photos and manually extracted metadata for MOV files. This ensures accurate organization by creation date.

Determining if a file is a duplicate is done by hashing the file contents. It's called a fingerprint inside the application. If the hash is the same, the files are considered duplicates. Hashing is done on only a part of a file. Since the tool supports MOV files and they can easily be 10G hashing the whole file is not practical even though it provides the most certainty. The tool will hash the first and last N MB of the file. This should be enough to determine if the files are the same. The amount of megabytes to use is determined by the size of the file. Minimum being 1 MB and maximum 10 MB. The only time I see this approach causing issues is if photos were taken in a studio setting with exactly the same lighting setup at a really high capture rate.

The metadata of a media file is in the beginning of the file so hashing the start and end of the file would include the metadata as well. This makes it not necessary to add parameters to the fingerprinting process.

To determine the sorting of the files the tool will read the metadata. This is done by reading the EXIF data from jpg and raf files. For mov files, the metadata is extracted manually. The tool will sort the files by the creation date of the media.

## testing

For testing a black box approach was taken. To reduce the amount of testing code needed the tests describe the behaviors that the application needs to fulfill.

## TODO

- add compare-only mode that would not move the files, but could be used to just validate that both directories contain the same data
