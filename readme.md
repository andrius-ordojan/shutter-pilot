# shutter-pilot

Program to organize media files. I like the structure that Lighroom uses when I import my photos and wanted a similar solution for when I'm on the go or backing up content out of the camera. This allows me to dump my photos and videos into an ingestion folder, then run the tool to read the metadata and organize media into folders based on the creation date.

WHAT PROBLEM DOES THIS SOLVE?

The file structure looks like This

TURN INTO PHOTO

- DESTINATION that was listed in the cli argument
  - `photos`
    - year ex. `2024`
      - full ISO date ex. `2024-11-10`
        - `raf` files go here. File names are preserved.
        - `sooc`
          - 'jpg' files are saved here with original name
  - `videos`
    - year
      - full ISO date
        - file with original name

## installation

Grab a binary from the releases.

or install using go install

## usage

```
mediaarrangelr [--dryrun] SOURCE DESTINATION
```

Setting `dryrun` will not run any changes on the file system.

## design and implementation decisions

## TODO

- [ ] rewrite the github page summery and the readme
- [ ] release new version

- add compare-only mode that would no move the files, but could be used to just validate that both directories contain the same data
