package processes

import (
	"cmp"
	"slices"
	"strings"
)

// The processes among allProcesses that have the same current working directory
// as proc, sorted by command name and then by PID.
//
// cwds maps PIDs to working directories, see GetCwdsByPid(). Those PIDs come
// without names, so allProcesses is what turns them back into processes we can
// name and sort by; pass every process you know about. PIDs we find no
// candidate for are dropped, since we would have nothing to call them.
//
// Candidates missing from cwds are skipped: they are either processes we aren't
// allowed to inspect, or ones that started or died in between the process
// listing and the cwd listing.
//
// The result is empty if proc itself is missing from cwds, since we then have
// nothing to compare against. proc is never part of the result.
func CwdFriends(proc *Process, allProcesses []*Process, cwds map[int]string) []*Process {
	cwd, weKnowOurCwd := cwds[proc.Pid]
	if !weKnowOurCwd {
		return nil
	}

	var friends []*Process
	for _, candidate := range allProcesses {
		if candidate.Pid == proc.Pid {
			continue
		}

		candidateCwd, found := cwds[candidate.Pid]
		if !found {
			continue
		}

		if candidateCwd != cwd {
			continue
		}

		friends = append(friends, candidate)
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
