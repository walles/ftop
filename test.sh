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

echo
PLATFORM="$(go env GOOS)-$(go env GOARCH)"
GOARCH_32=""
if [ "$PLATFORM" == "linux-amd64" ]; then
  GOARCH_32="386"
elif [ "$PLATFORM" == "linux-arm64" ]; then
  GOARCH_32="arm"
fi
if [ -n "$GOARCH_32" ]; then
  echo "Running unit tests on 32-bit architecture ($GOARCH_32)..."
  GOARCH="$GOARCH_32" go test -timeout 20s ./...
else
  echo "Skipping unit tests on 32-bit architecture since GOARCH=$(go env GOARCH)"
fi

# Try cross compiling
GOOS=linux GOARCH=386 ./build.sh

echo
echo "All tests passed!"
