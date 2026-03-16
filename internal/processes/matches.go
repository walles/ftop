package processes

import (
	"sort"
	"time"

	"github.com/walles/ftop/internal/log"
)

type ProcessMatch struct {
	Old *Process
	New *Process
}

type ProcessMatching struct {
	Matched      map[int]ProcessMatch
	Gone         []*Process
	New          []*Process
	CurrentByPid map[int]*Process
}

// Return child PIDs as a sorted slice
func childPids(proc *Process) []int {
	childPids := make([]int, 0, len(proc.children))
	for _, child := range proc.children {
		childPids = append(childPids, child.Pid)
	}

	sort.Ints(childPids)
	return childPids
}

// Match processes between previous and current frames, categorizing them as
// matched, gone, or new.
func buildProcessMatches(previous, current map[int]*Process) ProcessMatching {
	matching := ProcessMatching{
		Matched:      make(map[int]ProcessMatch, len(previous)),
		Gone:         make([]*Process, 0),
		New:          make([]*Process, 0),
		CurrentByPid: current,
	}

	for pid, oldProc := range previous {
		newProc, found := current[pid]
		if !found {
			matching.Gone = append(matching.Gone, oldProc)
			continue
		}

		if !oldProc.SameAs(newProc) {
			log.Infof(
				"SameAs() mismatch:\n  old=%s startTime=%s ppid=%d children=%v\n  new=%s startTime=%s ppid=%d children=%v\n  delta=%s",
				oldProc.String(),
				oldProc.startTime.Format(time.RFC3339Nano),
				oldProc.ppid,
				childPids(oldProc),
				newProc.String(),
				newProc.startTime.Format(time.RFC3339Nano),
				newProc.ppid,
				childPids(newProc),
				newProc.startTime.Sub(oldProc.startTime).Abs(),
			)

			matching.Gone = append(matching.Gone, oldProc)
			matching.New = append(matching.New, newProc)
			continue
		}

		matching.Matched[pid] = ProcessMatch{Old: oldProc, New: newProc}
	}

	for pid, newProc := range current {
		if _, found := previous[pid]; found {
			continue
		}

		matching.New = append(matching.New, newProc)
	}

	return matching
}
