# media-arrange-lr

Program to organize media files. I like the structure that Lighroom uses when I import my photos and wanted a similar solution for when I'm on the go or backing up content out of the camera. This allows me to dump my photos and videos into an ingestion folder, then run the tool to read the metadata and organize media into folders based on the creation date and move them to the final destination.

## installation

Grab a binary from the releases.

## usage

```
mediaarrangelr [--dryrun] SOURCE DESTINATION
```

Setting `dryrun` will not run any changes on the file system.
