package processes

import (
	"fmt"
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

type mismatchEntry struct {
	old   *Process
	new   *Process
	delta time.Duration
}

// Match processes between previous and current frames, categorizing them as
// matched, gone, or new.
func buildProcessMatches(previous, current map[int]*Process) (ProcessMatching, error) {
	matching := ProcessMatching{
		Matched:      make(map[int]ProcessMatch, len(previous)),
		Gone:         make([]*Process, 0),
		New:          make([]*Process, 0),
		CurrentByPid: current,
	}

	var mismatches []mismatchEntry

	for pid, oldProc := range previous {
		newProc, found := current[pid]
		if !found {
			matching.Gone = append(matching.Gone, oldProc)
			continue
		}

		if !oldProc.SameAs(newProc) {
			delta := newProc.startTime.Sub(oldProc.startTime).Abs()
			mismatches = append(mismatches, mismatchEntry{old: oldProc, new: newProc, delta: delta})
			continue
		}

		matching.Matched[pid] = ProcessMatch{Old: oldProc, New: newProc}
	}

	// If two or more mismatches share the same delta fraction, this indicates
	// that we (ftop) were paused while getting the snapshot, rather than actual
	// PID reuse. Treat those as matched rather than real PID reuse.
	bySubsecondDelta := make(map[time.Duration][]int, len(mismatches))
	for i, m := range mismatches {
		key := m.delta % time.Second
		bySubsecondDelta[key] = append(bySubsecondDelta[key], i)
	}

	for _, indices := range bySubsecondDelta {
		if len(indices) > 1 {
			return ProcessMatching{}, timeAnomalyError(fmt.Errorf(
				"%d mismatches share the same sub-second delta, indicating a time anomaly",
				len(indices)))
		}
	}

	for _, m := range mismatches {
		log.Infof(
			"SameAs() mismatch:\n  old=%s startTime=%s ppid=%d children=%v\n  new=%s startTime=%s ppid=%d children=%v\n  delta=%s",
			m.old.String(),
			m.old.startTime.Format(time.RFC3339Nano),
			m.old.ppid,
			childPids(m.old),
			m.new.String(),
			m.new.startTime.Format(time.RFC3339Nano),
			m.new.ppid,
			childPids(m.new),
			m.delta,
		)
		matching.Gone = append(matching.Gone, m.old)
		matching.New = append(matching.New, m.new)
	}

	for pid, newProc := range current {
		if _, found := previous[pid]; found {
			continue
		}

		matching.New = append(matching.New, newProc)
	}

	return matching, nil
}
