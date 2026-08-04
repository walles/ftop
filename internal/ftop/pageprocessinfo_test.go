package ftop

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

// Replaces the lsof socket lookup for the duration of the test. The returned
// function tells how many times it has been called.
func fakeSockets(t *testing.T, socketsByPid map[int][]processes.Socket, err error) func() int {
	t.Helper()

	original := getSocketsByPid
	t.Cleanup(func() {
		getSocketsByPid = original
	})

	calls := 0
	getSocketsByPid = func() (map[int][]processes.Socket, error) {
		calls++
		return socketsByPid, err
	}

	return func() int {
		return calls
	}
}

// Both connection sections render from one and the same socket listing, so that
// they can't disagree about a connection that came or went in between two lsof
// runs.
func TestWriteProcessInfoListsSocketsOnce(t *testing.T) {
	fakeCwds(t, map[int]string{42: "/Users/johan/src/ftop"}, nil)
	socketCalls := fakeSockets(t, map[int][]processes.Socket{}, nil)
	fakeDns(t, nil)

	original := getLoggedInUsersAt
	t.Cleanup(func() {
		getLoggedInUsersAt = original
	})
	getLoggedInUsersAt = func(time.Time) ([]string, error) {
		return []string{"alice"}, nil
	}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder

	ui.writeProcessInfo(&processes.Process{Pid: 42, Cmdline: "picked"}, nil, &page)

	assert.Equal(t, socketCalls(), 1)
}

// Every section is preceded by exactly two empty lines, and the closing
// separator by exactly one. Sections are responsible for their own titles, so
// the page is only readable if the space between them is added exactly once.
func TestWriteProcessInfoSeparatesSections(t *testing.T) {
	fakeCwds(t, map[int]string{42: "/Users/johan/src/ftop"}, nil)
	fakeSockets(t, map[int][]processes.Socket{}, nil)
	fakeDns(t, nil)

	original := getLoggedInUsersAt
	t.Cleanup(func() {
		getLoggedInUsersAt = original
	})
	getLoggedInUsersAt = func(time.Time) ([]string, error) {
		return []string{"alice"}, nil
	}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder

	ui.writeProcessInfo(&processes.Process{Pid: 42, Cmdline: "picked"}, nil, &page)

	// Titles are the only lines with the horizontal bar in them
	var titleIndices []int
	lines := strings.Split(page.String(), "\n")
	for i, line := range lines {
		if strings.Contains(line, "──") {
			titleIndices = append(titleIndices, i)
		}
	}

	// One per section, plus the closing separator
	assert.Equal(t, len(titleIndices), 9)

	// writeTitle() ends with an empty line of its own, so the first section's
	// title is the only one not preceded by blank space.
	assert.Equal(t, titleIndices[0], 0)

	for _, titleIndex := range titleIndices[1 : len(titleIndices)-1] {
		assert.Equal(t, lines[titleIndex-1], "")
		assert.Equal(t, lines[titleIndex-2], "")
		assert.Equal(t, lines[titleIndex-3] == "", false)
	}

	closingSeparator := titleIndices[len(titleIndices)-1]
	assert.Equal(t, lines[closingSeparator-1], "")
	assert.Equal(t, lines[closingSeparator-2] == "", false)
}

// Composing can crash halfway through, and all the pager would otherwise show
// is a page that stops for no stated reason. The note has to make it into the
// page before the pipe closes, or the reader never sees it.
func TestComposeProcessInfoPutsCrashesInThePage(t *testing.T) {
	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")

	pipeReader, pipeWriter := io.Pipe()

	// A nil process crashes the composer as soon as it looks at it
	go ui.composeProcessInfo(nil, nil, pipeWriter)

	page, err := io.ReadAll(pipeReader)
	assert.Equal(t, err, nil)
	assert.Equal(t, strings.Contains(string(page), "<Page composition crashed:"), true)
}
