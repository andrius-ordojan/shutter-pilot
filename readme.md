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

## Installation

Shutter-Pilot can be installed in two ways: by downloading a prebuilt binary or building it from source. Follow the instructions below to get started.

### Option 1: download binary

1. Go to the [Releases Page](https://github.com/andrius-ordojan/shutter-pilot/releases).
2. Download the binary for your operating system.
3. Place the binary in a directory listed in your `PATH`.
4. Verify the installation by running the command.

```bash
shutter-pilot --help
```

### Option 2: build from source

Use go install to build and install the binary

1. Ensure you have go installed on your system.
2. Use go install to build and install the binary.

```bash
go install github.com/andrius-ordojan/shutter-pilot@main
```

3. Make sure the go bin directory is in your PATH.

```bash
export PATH=$PATH:$(go env GOPATH)/bin

```

4. Verify the installation by running the command.

```bash
shutter-pilot --help
```

## Usage

```
Compares media files in source directories with destination directory and organises them
Usage: shutter-pilot [--filter FILTER] [--move] [--dryrun] [--nosooc] SOURCES DESTINATION

Positional arguments:
SOURCES source directories for media. Provide as a comma-separated list, e.g., /path/1,/path2/
DESTINATION destination directory for orginised media

Options:
--filter FILTER, -f FILTER
Filter by file types (allowed: jpg, raf, mov). Provide as a comma-separated list, e.g., -f jpg,mov
--move, -m moves files instead of copying [default: false]
--dryrun, -d does not modify file system [default: false]
--nosooc, -s Does no place jpg photos under sooc directory, but next to raw files [default: false]
--help, -h display this help and exit

```

## Examples

#### Basic Usage

Compare source and destination directories. Copy the missing files to the destination directory and organize them:

```bash
shutter-pilot /path/to/source /path/to/destination
```

#### Multiple Source Directories

Provide multiple source directories as a comma-separated list:

```bash
shutter-pilot /path/to/source1,/path/to/source2 /path/to/destination
```

#### Dry Run Mode

Preview changes without making any modifications:

```bash
shutter-pilot --dryrun /path/to/source /path/to/destination
```

#### Filter by File Types

Handle only JPG files and avoid using the "sooc" subdirectory. This will place jpg files under the date folder:

```bash
shutter-pilot --filter jpg --nosooc /path/to/source /path/to/destination
```

#### Filter Multiple File Types

Specify multiple file types to process:

```bash
shutter-pilot --filter jpg,raf /path/to/source /path/to/destination
```

### File conflicts

If duplicate files are found in the destination directory (based on hash), Shutter-Pilot will stop and report the conflicts. These must be resolved manually before proceeding. The tool does not make decisions on how to handle these situations.

## How it works

Shutter-Pilot uses a combination of file hashing and metadata extraction to compare, organize, and sort media files effectively.

### File Comparison

To determine if files are duplicates, Shutter-Pilot computes a **fingerprint** of each file by hashing its content. This approach ensures accuracy regardless of filenames or metadata. Since large files (like MOV) can be gigabytes in size, hashing is optimized as follows:

- **Partial Hashing**: Only the first and last segments of the file (1–10 MB) are hashed.
- **Dynamic Segment Size**: The size of each segment is proportional to the file size, with a minimum of 1 MB and a maximum of 10 MB.

This method balances performance and accuracy, ensuring that even large media files can be processed efficiently. The only edge case might occur when identical files are captured under studio conditions with identical metadata and content.

### File Organization

To determine the sorting of the files the tool will read the metadata. This is done by reading the EXIF data from JPG and RAF files. For MOV files, the metadata is extracted manually. The tool will sort the files by the creation date of the media.

### Stateless Operation

Each run is independent, with no reliance on external databases or persistent state.

## Testing

Shutter-Pilot uses a black box testing approach to verify its functionality from an end-user perspective. This ensures all core features behave as expected.

## To do

- Add a "compare-only" mode to validate that source and destination directories contain the same data without moving files.
