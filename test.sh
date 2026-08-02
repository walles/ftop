#!/bin/bash

set -e -o pipefail

# Start by verifying we can build
echo Building sources...
./build.sh

# Linting
echo
echo 'Linting, repro any errors locally using "golangci-lint run"...'
echo '  Linting without tests...'
golangci-lint run --tests=false
echo '  Linting with tests...'
golangci-lint run --tests=true

# Unit tests
echo
echo "Running unit tests..."
RACE=-race
if [ "$(go env GOARCH)" == "386" ]; then
  # -race is not supported on i386
  RACE=""
fi
go test $RACE -timeout 20s ./...

# Ensure we can build for all platforms we release for
#
# NOTE: Keep this list in sync with the release builds in release.sh
echo
echo "Testing cross compilation..."
echo "  Linux i386..."
GOOS=linux GOARCH=386 ./build.sh
echo "  Linux arm32..."
GOOS=linux GOARCH=arm ./build.sh

# The macOS binaries get their system stats through CGO, and cross compiling
# CGO code for macOS requires the macOS SDK. So these builds only work on macOS.
if [ "$(go env GOHOSTOS)" = "darwin" ]; then
  echo "  macOS amd64..."
  CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 ./build.sh
  echo "  macOS arm64..."
  CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 ./build.sh
else
  echo "  Skipping the macOS builds, they need the macOS SDK for CGO"
fi

echo
echo "All tests passed!"
