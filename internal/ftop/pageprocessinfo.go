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
	bottomUpNames := make([]string, 0)
	for p := proc; p != nil; p = p.Parent() {
		bottomUpNames = append(bottomUpNames, p.Command)
	}
	maxDepth := len(bottomUpNames) - 1
	treeLines := make([]string, len(bottomUpNames))
	for i, name := range bottomUpNames {
		depth := maxDepth - i
		indent := strings.Repeat(" ", depth)
		switch depth {
		case 0:
			treeLines[depth] = name
		case maxDepth:
			// Current process: nothing branches below, so no downward connector
			treeLines[depth] = indent + "└── " + name
		default:
			// └┬─: up-and-right corner, then down-and-horizontal (┬) so the
			// next └ below aligns under the ┬ rather than a plain horizontal bar
			treeLines[depth] = indent + "└┬─ " + name
		}
	}
	lines += "\n\n" + strings.Join(treeLines, "\n")

	return moor.PageFromString(lines, moor.Options{NoLineNumbers: true})
}
