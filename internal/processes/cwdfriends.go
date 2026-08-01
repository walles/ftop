package processes

import (
	"cmp"
	"slices"
	"strings"
)

// The processes among others that have the same current working directory as
// proc, sorted by command name and then by PID.
//
// cwds maps PIDs to working directories, see GetCwdsByPid(). Processes missing
// from cwds are never returned, and if proc itself is missing the result is
// empty since we then have nothing to compare against.
//
// proc is never part of the result, not even if it is also in others.
func CwdFriends(proc *Process, others []*Process, cwds map[int]string) []*Process {
	cwd, weKnowOurCwd := cwds[proc.Pid]
	if !weKnowOurCwd {
		return nil
	}

	var friends []*Process
	for _, other := range others {
		if other.Pid == proc.Pid {
			continue
		}

		otherCwd, found := cwds[other.Pid]
		if !found {
			continue
		}

		if otherCwd != cwd {
			continue
		}

		friends = append(friends, other)
	}

	// Login shells are launched as "-bash" or "-fish". Drop the dash so that
	// they sort next to their non-login siblings.
	sortKey := func(p *Process) string {
		return strings.TrimPrefix(p.Command(), "-")
	}

	slices.SortFunc(friends, func(a, b *Process) int {
		return cmp.Or(
			strings.Compare(sortKey(a), sortKey(b)),
			cmp.Compare(a.Pid, b.Pid),
		)
	})

	return friends
}
