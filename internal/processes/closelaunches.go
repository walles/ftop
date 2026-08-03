package processes

import (
	"slices"
)

// The processes launched closest in time to proc, ordered by start time.
//
// Candidates are the other processes in proc's own tree, found by walking up
// from proc to the root and back down again. proc is never part of the result.
//
// The result holds the five closest launches, plus the nearest launch before
// and the nearest one after proc when those aren't already among the five. So
// at most seven, and fewer when proc's tree is smaller than that. It is empty
// only if proc is alone in its tree.
func CloseLaunches(proc *Process) []*Process {
	allOtherProcs := allOtherProcessesInTree(proc)

	// Sort by launch time closeness
	zero := proc.StartTime()
	slices.SortFunc(allOtherProcs, func(a, b *Process) int {
		diffA := a.StartTime().Sub(zero).Abs().Milliseconds()
		diffB := b.StartTime().Sub(zero).Abs().Milliseconds()

		if diffA == diffB {
			return 0
		}
		if diffA < diffB {
			return -1
		}

		return 1
	})

	// Extract the five closest launches. Without cloning, our re-sort (below)
	// of the allOtherProcs list messes up the contents of topList as well.
	topList := slices.Clone(allOtherProcs[:min(5, len(allOtherProcs))])

	// Find one before and one after process
	sortByStartTime(allOtherProcs)
	var before, after *Process
	for _, candidate := range allOtherProcs {
		if candidate.StartTime().Before(zero) {
			before = candidate
		} else if candidate.StartTime().After(zero) && after == nil {
			after = candidate
			break
		}
	}
	if before != nil && !slices.ContainsFunc(topList, func(p *Process) bool { return p.SameAs(before) }) {
		topList = append(topList, before)
	}
	if after != nil && !slices.ContainsFunc(topList, func(p *Process) bool { return p.SameAs(after) }) {
		topList = append(topList, after)
	}

	sortByStartTime(topList)

	return topList
}

func sortByStartTime(procs []*Process) {
	slices.SortFunc(procs, func(a, b *Process) int {
		if a.StartTime().Before(b.StartTime()) {
			return -1
		}
		if a.StartTime().After(b.StartTime()) {
			return 1
		}

		return 0
	})
}

// All processes in proc's tree except proc itself, in no particular order.
func allOtherProcessesInTree(proc *Process) []*Process {
	// Find the root process
	init := proc
	for init.Parent() != nil {
		init = init.Parent()
	}

	// Flatten the process tree
	allProcs := []*Process{}
	var flatten func(p *Process)
	flatten = func(p *Process) {
		if !p.SameAs(proc) {
			allProcs = append(allProcs, p)
		}

		for _, child := range p.Children() {
			flatten(child)
		}
	}
	flatten(init)

	return allProcs
}
