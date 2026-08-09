package ftop

import (
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/walles/ftop/internal/assert"
)

// The pager renders lines as they arrive, which is only useful if we hand them
// over as we compose them. Buffering the page up until the last section is done
// would make the slow sections hold up the fast ones.
func TestPageTextWritesThrough(t *testing.T) {
	pipeReader, pipeWriter := io.Pipe()
	pt := pageText{out: pipeWriter}

	go func() {
		pt.writeLine("early")

		// Intentionally leaving the pipe open: the point is that "early" is
		// readable while composition is still going on.
	}()

	early := make([]byte, len("early\n"))
	_, err := io.ReadFull(pipeReader, early)
	if err != nil {
		t.Fatalf("Reading the first line: %v", err)
	}

	assert.Equal(t, string(early), "early\n")
}

// Everything above the first blank line, unstyled: the title and its border.
func titleLine(page string) string {
	line, _, _ := strings.Cut(stripAnsi(page), "\n")
	return line
}

// A title fills 80 columns, border included, so that the titles of a page's
// several sections all end in the same place.
func TestWriteTitleWidth(t *testing.T) {
	var page strings.Builder
	pt := pageText{out: &page}

	pt.writeTitle("Launch Hierarchy")

	assert.Equal(t, titleLine(page.String()), "──Launch Hierarchy"+strings.Repeat("─", 62))
}

// The border is measured in terminal columns rather than in characters, so a
// title naming a process with a CJK name — two columns per character — gets a
// correspondingly shorter border instead of one running past 80 columns.
func TestWriteTitleWidthWithWideCharacters(t *testing.T) {
	var page strings.Builder
	pt := pageText{out: &page}

	pt.writeTitle("写真整理(42)")

	assert.Equal(t, titleLine(page.String()), "──写真整理(42)"+strings.Repeat("─", 66))
}

func stringsContains(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}

// Matches SGR escape sequences, which is everything twin emits. Improve this if
// we ever start emitting anything else.
var sgrSequence = regexp.MustCompile("\x1b\\[[0-9;]*m")

// Page text without any of the styling, for comparing against expected layouts.
func stripAnsi(styled string) string {
	return sgrSequence.ReplaceAllString(styled, "")
}

// Everything a section wrote below its own title, unstyled.
//
// Section titles end in a blank line and nothing else in a section does, so the
// first one is where the title ends and the body begins.
//
// Compare this against a complete expected block rather than picking fragments
// out of it with stringsContains(): alignment is a feature of these sections, and
// asserting "sshd(123)" passes whether or not the columns line up. An expected
// block doubles as documentation of what a section looks like, which is why the
// mockups the design notes used to carry are now these strings. stringsContains()
// is for error and empty states, where there is no layout to get wrong.
func sectionBody(page string) string {
	_, body, _ := strings.Cut(stripAnsi(page), "\n\n")
	return body
}
