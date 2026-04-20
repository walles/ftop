package processes

import (
	"testing"
	"time"

	"github.com/walles/ftop/internal/assert"
)

func TestBuildProcessMatches_IgnoresGlobalShiftWithIdenticalSubsecondDelta(t *testing.T) {
	base := time.Date(2026, time.April, 20, 12, 0, 0, 0, time.UTC)

	old1 := &Process{Pid: 101, startTime: base, Cmdline: "cmd1"}
	old2 := &Process{Pid: 102, startTime: base.Add(10 * time.Second), Cmdline: "cmd2"}
	old3 := &Process{Pid: 103, startTime: base.Add(20 * time.Second), Cmdline: "cmd3"}
	old4 := &Process{Pid: 104, startTime: base.Add(30 * time.Second), Cmdline: "cmd4"}

	previous := map[int]*Process{
		old1.Pid: old1,
		old2.Pid: old2,
		old3.Pid: old3,
		old4.Pid: old4,
	}

	// new1/new2 share the same sub-second delta and should be treated as a
	// global-shift anomaly. new3 has a different sub-second delta and should
	// not.
	new1 := &Process{Pid: 101, startTime: old1.startTime.Add(30*time.Minute + 123456789*time.Nanosecond), Cmdline: "cmd1"}
	new2 := &Process{Pid: 102, startTime: old2.startTime.Add(45*time.Minute + 123456789*time.Nanosecond), Cmdline: "cmd2"}
	new3 := &Process{Pid: 103, startTime: old3.startTime.Add(30*time.Minute + 234567890*time.Nanosecond), Cmdline: "cmd3"}
	new4 := &Process{Pid: 104, startTime: old4.startTime, Cmdline: "cmd4"}

	current := map[int]*Process{
		new1.Pid: new1,
		new2.Pid: new2,
		new3.Pid: new3,
		new4.Pid: new4,
	}

	matches := buildProcessMatches(previous, current)

	assert.Equal(t, len(matches.Matched), 3)
	assert.Equal(t, len(matches.Gone), 1)
	assert.Equal(t, len(matches.New), 1)

	assert.Equal(t, matches.Matched[101].Old, old1)
	assert.Equal(t, matches.Matched[101].New, new1)
	assert.Equal(t, matches.Matched[102].Old, old2)
	assert.Equal(t, matches.Matched[102].New, new2)
	assert.Equal(t, matches.Matched[104].Old, old4)
	assert.Equal(t, matches.Matched[104].New, new4)

	assert.Equal(t, matches.Gone[0], old3)
	assert.Equal(t, matches.New[0], new3)
}
