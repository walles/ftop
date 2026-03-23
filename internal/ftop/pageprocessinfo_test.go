package ftop

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

func TestUsersLoggedInWhenProcessStartedForPaging(t *testing.T) {
	original := getLoggedInUsersAt
	t.Cleanup(func() {
		getLoggedInUsersAt = original
	})

	getLoggedInUsersAt = func(time.Time) ([]string, error) {
		return []string{"alice", "bob from 10.0.0.5"}, nil
	}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	pt := pageText{}

	ui.usersLoggedInWhenProcessStartedForPaging(&processes.Process{Pid: 42, Command: "picked"}, &pt)

	assert.Equal(t, stringsContains(pt.String(), "Users logged in when picked(42) started"), true)
	assert.Equal(t, stringsContains(pt.String(), "\nalice\n"), true)
	assert.Equal(t, stringsContains(pt.String(), "\nbob from 10.0.0.5\n"), true)
	assert.Equal(t, stringsContains(pt.String(), "  alice"), false)
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
	pt := pageText{}

	ui.usersLoggedInWhenProcessStartedForPaging(&processes.Process{Pid: 42, Command: "picked"}, &pt)

	assert.Equal(t, stringsContains(pt.String(), "\n<Unable to inspect login history: boom>\n"), true)
	assert.Equal(t, stringsContains(pt.String(), "\n  <Unable to inspect login history: boom>\n"), false)
}

func stringsContains(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}
