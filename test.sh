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
echo "All tests passed!"
