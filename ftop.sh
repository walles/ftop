#!/bin/bash

# Build ftop and run it, this script should behave just like the binary.

set -e -o pipefail

MYDIR="$(
    cd "$(dirname "$0")"
    pwd
)"
cd "$MYDIR"

rm -f ftop

RACE=-race ./build.sh 1>&2

GORACE="log_path=ftop-race-report" ./ftop "$@"
