package processes

import (
	"testing"

	"github.com/walles/ftop/internal/assert"
)

// Without knowing our own working directory we can't tell who shares it.
func TestCwdFriends_ownCwdUnknown(t *testing.T) {
	me := &Process{Pid: 123, Cmdline: "me"}
	other := &Process{Pid: 234, Cmdline: "other"}

	friends := CwdFriends(me, []*Process{other}, map[int]string{234: "/notroot"})

	assert.Equal(t, len(friends), 0)
}

// Only processes with the exact same working directory are friends. Processes
// somewhere else, and processes we can't inspect, are not.
func TestCwdFriends_findFriends(t *testing.T) {
	me := &Process{Pid: 123, Cmdline: "me"}
	friend := &Process{Pid: 234, Cmdline: "friend"}
	elsewhere := &Process{Pid: 666, Cmdline: "elsewhere"}
	secretive := &Process{Pid: 667, Cmdline: "secretive"}

	cwds := map[int]string{
		me.Pid:        "/notroot",
		friend.Pid:    "/notroot",
		elsewhere.Pid: "/somewhereelse",
		// secretive's working directory intentionally left out
	}

	friends := CwdFriends(me, []*Process{friend, elsewhere, secretive}, cwds)

	assert.SlicesEqual(t, friends, []*Process{friend})
}

// We are not our own friend, not even when we show up in the others list.
func TestCwdFriends_excludesSelf(t *testing.T) {
	me := &Process{Pid: 123, Cmdline: "me"}

	friends := CwdFriends(me, []*Process{me}, map[int]string{me.Pid: "/notroot"})

	assert.Equal(t, len(friends), 0)
}

// Friends are listed by command name with the PID breaking ties. The leading
// dash that login shells have does not affect the ordering.
func TestCwdFriends_ordering(t *testing.T) {
	me := &Process{Pid: 1000, Cmdline: "me"}
	awk := &Process{Pid: 4, Cmdline: "awk"}
	loginBash := &Process{Pid: 2, Cmdline: "-bash"}
	bash := &Process{Pid: 3, Cmdline: "bash"}
	laterCat := &Process{Pid: 6, Cmdline: "cat"}
	earlierCat := &Process{Pid: 5, Cmdline: "cat"}

	others := []*Process{bash, laterCat, awk, earlierCat, loginBash}

	cwds := map[int]string{me.Pid: "/shared"}
	for _, other := range others {
		cwds[other.Pid] = "/shared"
	}

	friends := CwdFriends(me, others, cwds)

	assert.SlicesEqual(t, friends, []*Process{awk, loginBash, bash, earlierCat, laterCat})
}
