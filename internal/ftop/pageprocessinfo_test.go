package ftop

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
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

func TestUsersLoggedInWhenProcessStartedForPaging(t *testing.T) {
	original := getLoggedInUsersAt
	t.Cleanup(func() {
		getLoggedInUsersAt = original
	})

	getLoggedInUsersAt = func(time.Time) ([]string, error) {
		return []string{"alice", "bob from 10.0.0.5"}, nil
	}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.usersLoggedInWhenProcessStartedForPaging(&processes.Process{Pid: 42, Cmdline: "picked"}, &pt)

	assert.Equal(t, stringsContains(page.String(), "Users logged in when picked(42) started"), true)
	assert.Equal(t, stringsContains(page.String(), "\nalice\n"), true)
	assert.Equal(t, stringsContains(page.String(), "\nbob from 10.0.0.5\n"), true)
	assert.Equal(t, stringsContains(page.String(), "  alice"), false)
}

func TestUsersLoggedInWhenProcessStartedForPagingShowsErrors(t *testing.T) {
	original := getLoggedInUsersAt
	t.Cleanup(func() {
		getLoggedInUsersAt = original
	})

	getLoggedInUsersAt = func(time.Time) ([]string, error) {
		return nil, errors.New("boom")
	}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.usersLoggedInWhenProcessStartedForPaging(&processes.Process{Pid: 42, Cmdline: "picked"}, &pt)

	assert.Equal(t, stringsContains(page.String(), "\n<Unable to inspect login history: boom>\n"), true)
	assert.Equal(t, stringsContains(page.String(), "\n  <Unable to inspect login history: boom>\n"), false)
}

// Replaces the lsof lookup for the duration of the test
func fakeCwds(t *testing.T, cwds map[int]string, err error) {
	t.Helper()

	original := getCwdsByPid
	t.Cleanup(func() {
		getCwdsByPid = original
	})

	getCwdsByPid = func() (map[int]string, error) {
		return cwds, err
	}
}

func TestCwdFriendsForPagingListsFriends(t *testing.T) {
	fakeCwds(t, map[int]string{
		42: "/Users/johan/src/ftop",
		7:  "/Users/johan/src/ftop",
		8:  "/Users/johan/src/ftop",
		9:  "/somewhereelse",
	}, nil)

	candidates := []*processes.Process{
		{Pid: 8, Cmdline: "gopls"},
		{Pid: 9, Cmdline: "elsewhere"},
		{Pid: 7, Cmdline: "fish"},
	}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.cwdFriendsForPaging(&processes.Process{Pid: 42, Cmdline: "picked"}, candidates, &pt)

	assert.Equal(t, stringsContains(page.String(), "\nfish(7)\ngopls(8)\n"), true)
	assert.Equal(t, stringsContains(page.String(), "elsewhere"), false)
}

func TestCwdFriendsForPagingShowsErrors(t *testing.T) {
	fakeCwds(t, nil, errors.New("boom"))

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.cwdFriendsForPaging(&processes.Process{Pid: 42, Cmdline: "picked"}, nil, &pt)

	assert.Equal(t, stringsContains(page.String(), "\n<Unable to list working directories: boom>\n"), true)
}

// lsof and our process listing are taken at slightly different times, so the
// picked process can be missing from the lsof output.
func TestCwdFriendsForPagingUnknownCwd(t *testing.T) {
	fakeCwds(t, map[int]string{1: "/somewhere"}, nil)

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.cwdFriendsForPaging(&processes.Process{Pid: 42, Cmdline: "picked"}, nil, &pt)

	assert.Equal(t, stringsContains(page.String(), "\n<Working directory unknown"), true)
}

// Half the system has / as its working directory, listing those is pointless.
func TestCwdFriendsForPagingRootCwd(t *testing.T) {
	fakeCwds(t, map[int]string{42: "/"}, nil)

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.cwdFriendsForPaging(&processes.Process{Pid: 42, Cmdline: "picked"}, nil, &pt)

	assert.Equal(t, stringsContains(page.String(), "Others sharing this process' working directory (/)"), true)
	assert.Equal(t, stringsContains(page.String(), "\n<Working directory too common, never mind>\n"), true)
}

func TestCwdFriendsForPagingNoFriends(t *testing.T) {
	fakeCwds(t, map[int]string{42: "/Users/johan/src/ftop"}, nil)

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.cwdFriendsForPaging(&processes.Process{Pid: 42, Cmdline: "picked"}, nil, &pt)

	assert.Equal(t, stringsContains(page.String(), "Others sharing this process' working directory (/Users/johan/src/ftop)"), true)
	assert.Equal(t, stringsContains(page.String(), "\n<Nobody else shares this working directory>\n"), true)
}

func stringsContains(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}
