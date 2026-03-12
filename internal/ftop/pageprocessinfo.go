package ftop

import (
	"strings"

	"github.com/walles/ftop/internal/processes"
	"github.com/walles/moor/v2/pkg/moor"
)

func pageProcessInfo(proc *processes.Process) error {
	if proc == nil {
		panic("proc is nil, can't page process info")
	}

	lines := strings.Join(proc.CommandLine(), "\n  ")

	// Build launch hierarchy from root down to current process
	bottomUpProcs := make([]*processes.Process, 0)
	for p := proc; p != nil; p = p.Parent() {
		bottomUpProcs = append(bottomUpProcs, p)
	}
	maxDepth := len(bottomUpProcs) - 1
	treeLines := make([]string, len(bottomUpProcs))
	for i, p := range bottomUpProcs {
		depth := maxDepth - i
		if depth == maxDepth {
			// Arrow replaces the indent; make it the same width so the name aligns
			arrow := strings.Repeat("-", max(0, 2*maxDepth-2)) + "> "
			treeLines[depth] = arrow + p.String()
		} else {
			treeLines[depth] = strings.Repeat("  ", depth) + p.String()
		}
	}
	lines += "\n\n" + strings.Join(treeLines, "\n")

	return moor.PageFromString(lines, moor.Options{NoLineNumbers: true})
}
