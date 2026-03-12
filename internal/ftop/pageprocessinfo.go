package ftop

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/util"
	"github.com/walles/moor/v2/pkg/moor"
	"github.com/walles/moor/v2/twin"
)

func (u *Ui) pageProcessInfo(proc *processes.Process) error {
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
			arrow := strings.Repeat("─", max(0, 2*maxDepth-2)) + "▶ "
			line = arrow + p.String()
		} else {
			line = strings.Repeat("  ", depth) + p.String()
		}
		entries[depth] = entry{line, p}
		maxWidth = max(maxWidth, utf8.RuneCountInString(line))
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
			maxWidth = max(maxWidth, utf8.RuneCountInString(line))
			appendChildren(child, depth+1)
		}
	}
	appendChildren(proc, maxDepth+1)

	treeLines := make([]string, len(entries))
	for i, e := range entries {
		padding := strings.Repeat(" ", maxWidth-utf8.RuneCountInString(e.line))
		treeLines[i] = e.line + padding + "  " + e.process.Username
	}
	lines += "\n\n" + strings.Join(treeLines, "\n")

	// Timing and CPU info for the current process, same as the main UI pane
	age := time.Since(proc.StartTime())
	cpuTime := proc.CpuTimeOrZero()
	percentCpu := 100.0 * float64(cpuTime) / float64(age)
	lines += fmt.Sprintf(
		"\n\nStarted %s ago at %s. It used %s CPU, or %s.",
		u.highlight(util.FormatDuration(age)),
		u.highlight(proc.StartTime().Format("2006-01-02 15:04:05")),
		u.highlight(util.FormatDuration(cpuTime)),
		u.highlight(util.FormatPercent(percentCpu)),
	)

	return moor.PageFromString(lines, moor.Options{NoLineNumbers: true})
}

func (u *Ui) highlight(s string) string {
	colored := twin.StyleDefault.WithForeground(u.theme.HighlightedForeground())
	notColored := twin.StyleDefault

	// "24 bit" is fine here, if the terminal doesn't support it, the pager will
	// just down sample it as needed.
	prefix := colored.RenderUpdateFrom(notColored, twin.ColorCount24bit)
	suffix := notColored.RenderUpdateFrom(colored, twin.ColorCount24bit)
	return prefix + s + suffix
}
