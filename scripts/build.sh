#!/bin/bash

# Set the output directory.
OUT_DIR="build"

mkdir -p $OUT_DIR

# Cross-compile for Mac.
env GOOS=darwin GOARCH=amd64 go build -v -x -o $OUT_DIR/spotify_export-mac64 cmd/spotify_export/main.go

# Cross-compile for Windows.
env GOOS=windows GOARCH=amd64 go build -v -x -o $OUT_DIR/spotify_export-win64.exe cmd/spotify_export/main.go

# Cross-compile for Linux.
env GOOS=linux GOARCH=amd64 go build -v -x -o $OUT_DIR/spotify_export-linux64 cmd/spotify_export/main.go
