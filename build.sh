#!/bin/bash

set -e

echo "Building for macOS (M1)..."
env GOOS=darwin GOARCH=arm64 go build -o build/sgpt-macos-m1

echo "Building for Windows (amd64)..."
env GOOS=windows GOARCH=amd64 go build -o build/sgpt-windows-amd64.exe

echo "Building for Linux (amd64)..."
env GOOS=linux GOARCH=amd64 go build -o build/sgpt-linux-amd64

echo "Build complete."