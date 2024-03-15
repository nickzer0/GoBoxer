#!/bin/bash

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "[!] Go not found, please install before running: https://go.dev/doc/install"
    exit 1
fi


# Attempt to build Go project
go build -o goboxer cmd/web/*.go 
echo "GoBoxer built successfully, run './goboxer' to start the server."