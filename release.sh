#!/bin/bash

set -e -o pipefail

# Bail if we're on a dirty version
if [ -n "$(git diff HEAD --stat)" ]; then
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

echo
echo "Building ${VERSION} release binaries..."

# NOTE: To get the version number right, production builds must be done after
# the above tagging.
GOOS=linux GOARCH=386 ./build.sh
GOOS=linux GOARCH=arm ./build.sh

# Certain macOS specific code is made using CGO
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 ./build.sh
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 ./build.sh

# Push the newly built release tag
echo
echo "Pushing ${VERSION} release tag..."
git push --no-verify --tags

# Create GitHub release with the binaries, using the tag annotation as the message
echo
echo "Creating GitHub release..."

TITLE="${VERSION}: ${FIRST_LINE}"

# shellcheck disable=SC2010  # Globbing can't replace "grep -v"
BINARIES=$(ls releases/ftop-"${VERSION}"-* | grep -v dirty | grep -v -- -g)

# shellcheck disable=SC2086  # We *want* BINARIES to be split here
gh release create "${VERSION}" \
    --title "$TITLE" \
    --notes "$DESCRIPTION" \
    --fail-on-no-commits \
    $BINARIES

echo
echo "Release ${VERSION} created successfully!"
echo "View it at: https://github.com/walles/ftop/releases/tag/${VERSION}"

# Update Homebrew tap
echo
echo "Updating Homebrew tap..."

# Calculate new tarball SHA256
#
# curl flags are: fail on HTTP errors, silent, show errors even when silent and follow redirects.
NEW_SHA256=$(curl -fsSL "https://github.com/walles/ftop/archive/refs/tags/${VERSION}.tar.gz" | shasum -a 256 | cut -d' ' -f1)

TAP_DIR="../homebrew-johan"
if [ -d "$TAP_DIR" ] && \
   [ -z "$(git -C "$TAP_DIR" diff HEAD --stat)" ] && \
   [ "$(git -C "$TAP_DIR" rev-parse --abbrev-ref HEAD)" = "main" ]; then
    # Make sure we are up to date
    git -C "$TAP_DIR" pull --quiet

    # All checks passed, update the formula
    FORMULA="$TAP_DIR/Formula/ftop.rb"
    sed -i '' "s|archive/refs/tags/v[0-9.]*\.tar\.gz|archive/refs/tags/${VERSION}.tar.gz|" "$FORMULA"
    sed -i '' "s|sha256 \"[a-f0-9]*\"|sha256 \"$NEW_SHA256\"|" "$FORMULA"

    # Commit and push
    git -C "$TAP_DIR" add Formula/ftop.rb
    git -C "$TAP_DIR" commit -m "ftop: update to ${VERSION}"
    git -C "$TAP_DIR" push

    echo "✓ Homebrew tap updated"
else
    echo "WARNING: Homebrew tap not updated."
    echo "  Please update this file manually: https://github.com/walles/homebrew-johan/blob/main/Formula/ftop.rb"
    echo "  - URL: https://github.com/walles/ftop/archive/refs/tags/${VERSION}.tar.gz"
    echo "  - SHA256: $NEW_SHA256"
fi
