package ftop

import (
	"errors"
	"strings"
	"testing"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

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
