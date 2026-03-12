package ftop

import (
	"slices"
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
	type entry struct {
		line    string
		process *processes.Process
	}
	entries := make([]entry, len(bottomUpProcs))
	maxWidth := 0

	for i, p := range bottomUpProcs {
		depth := maxDepth - i
		var line string
		if depth == maxDepth {
			arrow := strings.Repeat("-", max(0, 2*maxDepth-2)) + "> "
			line = arrow + p.String()
		} else {
			line = strings.Repeat("  ", depth) + p.String()
		}
		entries[depth] = entry{line, p}
		maxWidth = max(maxWidth, len(line))
	}

	// Append children of the current process recursively (sorted by command then PID, like px)
	var appendChildren func(p *processes.Process, depth int)
	appendChildren = func(p *processes.Process, depth int) {
		children := p.Children()
		slices.SortFunc(children, func(a, b *processes.Process) int {
			if a.Command != b.Command {
				return strings.Compare(strings.ToLower(a.Command), strings.ToLower(b.Command))
			}
			return a.Pid - b.Pid
		})
		for _, child := range children {
			line := strings.Repeat("  ", depth) + child.String()
			entries = append(entries, entry{line, child})
			maxWidth = max(maxWidth, len(line))
			appendChildren(child, depth+1)
		}
	}
	appendChildren(proc, maxDepth+1)

	treeLines := make([]string, len(entries))
	for i, e := range entries {
		padding := strings.Repeat(" ", maxWidth-len(e.line))
		treeLines[i] = e.line + padding + "  " + e.process.Username
	}
	lines += "\n\n" + strings.Join(treeLines, "\n")

	return moor.PageFromString(lines, moor.Options{NoLineNumbers: true})
}
