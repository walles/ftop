#!/bin/bash

set -e -o pipefail

# Bail if we're on a dirty version
if [ -n "$(git diff --stat)" ]; then
  echo "ERROR: Please commit all changes before doing a release"
  echo
  git status

  exit 1
fi

echo "Running tests before making the release..."
./test.sh

# Ask user to consider updating the screenshot
cat <<EOM

Please consider updating the screenshot in README.md before releasing.

Scale your terminal to 90x30, then run: go run ./cmd/ftop

Answer yes at this prompt to verify that the screenshot is up to date.
EOM

read -r -p "Screenshot up to date: " MAYBE_YES
if [ "$MAYBE_YES" != "yes" ]; then
  echo
  echo "Please update the screenshot, then try this script again."
  exit 0
fi

# List existing version numbers...
echo
echo "Previous version numbers:"
git tag | sort -V | tail

# ... and ask for a new version number.
echo
echo "Please provide a version number on the form 'v1.2.3' for the new release:"
read -r VERSION

if ! echo "${VERSION}" | grep -q -E '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "ERROR: Version number must be on the form: v1.2.3: ${VERSION}"
  exit 1
fi

# List changes since last release as inspiration...
LAST_VERSION="$(git describe --abbrev=0)"
echo

echo "Changes since last release:"
git log --first-parent --pretty="format:* %s" "${LAST_VERSION}"..HEAD | sed 's/ diff.*//'
echo

# Make an annotated tag for this release
echo
echo "Please provide a tag message for version ${VERSION}."
echo "The first line will be the release title."
echo "The second line must be empty."
echo "Subsequent lines will be the release description."
git tag --annotate "${VERSION}"

TAG_MESSAGE=$(git tag -l --format='%(contents)' "${VERSION}")
FIRST_LINE=$(echo "$TAG_MESSAGE" | sed -n '1p')
SECOND_LINE=$(echo "$TAG_MESSAGE" | sed -n '2p')
DESCRIPTION=$(echo "$TAG_MESSAGE" | tail -n +3)

if [ -z "$FIRST_LINE" ]; then
    echo "ERROR: First line of tag message must be the release title"
    git tag --delete "${VERSION}"
    exit 1
fi

if [ -n "$SECOND_LINE" ]; then
    echo "ERROR: Second line of tag message must be empty"
    git tag --delete "${VERSION}"
    exit 1
fi

# NOTE: To get the version number right, production builds must be done after
# the above tagging.
GOOS=linux GOARCH=386 ./build.sh
GOOS=linux GOARCH=arm ./build.sh

# Certain macOS specific code is made using CGO
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 ./build.sh
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 ./build.sh

# Push the newly built release tag
git push --tags

# Create GitHub release with the binaries, using the tag annotation as the message
echo
echo "Creating GitHub release..."

TITLE="${VERSION}: ${FIRST_LINE}"

# shellcheck disable=SC2010  # Globbing can't replace "grep -v"
BINARIES=$(ls releases/ftop-"${VERSION}"-* | grep -v dirty | grep -v -- -g)

if [ -n "$DESCRIPTION" ]; then
  # shellcheck disable=SC2086  # We want word splitting here
  gh release create "${VERSION}" \
    --title "$TITLE" \
    --notes "$DESCRIPTION" \
    --fail-on-no-commits \
    $BINARIES
else
  # shellcheck disable=SC2086  # We want word splitting here
  gh release create "${VERSION}" \
    --title "$TITLE" \
    --fail-on-no-commits \
    $BINARIES
fi

echo
echo "Release ${VERSION} created successfully!"
echo "View it at: https://github.com/walles/ftop/releases/tag/${VERSION}"
