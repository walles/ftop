package ftop

import (
	"io"
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

func stringsContains(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}
