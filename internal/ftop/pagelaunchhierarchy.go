package ftop

import (
	"slices"
	"strings"

	"github.com/rivo/uniseg"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/util"
	"github.com/walles/moor/v2/twin"
)

func (u *Ui) launchHierarchyForPaging(proc *processes.Process, pt *pageText) {
	pt.writeTitle("Launch Hierarchy")

	// Build launch hierarchy from root down to current process
	bottomUpProcs := make([]*processes.Process, 0)
	for p := proc; p != nil; p = p.Parent() {
		bottomUpProcs = append(bottomUpProcs, p)
	}
	maxDepth := len(bottomUpProcs) - 1
	type entry struct {
		line      string
		fancyLine string
		process   *processes.Process
	}
	entries := make([]entry, len(bottomUpProcs))
	maxWidth := 0

	for i, p := range bottomUpProcs {
		depth := maxDepth - i
		var line string
		var fancyLine string
		if depth == maxDepth {
			arrow := strings.Repeat("─", max(0, 2*maxDepth-2)) + "▶ "
			line = arrow + p.String()
			fancyLine = arrow + u.highlight(p.String())
		} else {
			line = strings.Repeat("  ", depth) + p.String()
			fancyLine = line
		}
		entries[depth] = entry{line, fancyLine, p}
		maxWidth = max(maxWidth, uniseg.StringWidth(line))
	}

	// Append children of the current process recursively (sorted by command then PID, like px)
	var appendChildren func(p *processes.Process, depth int)
	appendChildren = func(p *processes.Process, depth int) {
		children := p.Children()
		slices.SortFunc(children, func(a, b *processes.Process) int {
			if a.Command() != b.Command() {
				return strings.Compare(strings.ToLower(a.Command()), strings.ToLower(b.Command()))
			}

			return a.Pid - b.Pid
		})
		for _, child := range children {
			line := strings.Repeat("  ", depth) + child.String()
			entries = append(entries, entry{line, line, child})
			maxWidth = max(maxWidth, uniseg.StringWidth(line))
			appendChildren(child, depth+1)
		}
	}
	appendChildren(proc, maxDepth+1)

	currentUsername := util.GetCurrentUsername()

	treeLines := make([]string, len(entries))

	dim := twin.StyleDefault.WithAttr(twin.AttrDim)
	dimPrefix := dim.RenderUpdateFrom(twin.StyleDefault, twin.ColorCount24bit)
	dimSuffix := twin.StyleDefault.RenderUpdateFrom(dim, twin.ColorCount24bit)

	bold := twin.StyleDefault.WithAttr(twin.AttrBold)
	boldPrefix := bold.RenderUpdateFrom(twin.StyleDefault, twin.ColorCount24bit)
	boldSuffix := twin.StyleDefault.RenderUpdateFrom(bold, twin.ColorCount24bit)

	for i, e := range entries {
		username := e.process.Username
		switch username {
		case currentUsername:
			// This block intentionally left blank
		case "root":
			username = dimPrefix + username + dimSuffix
		default:
			username = boldPrefix + username + boldSuffix
		}

		treeLines[i] = rightPadded(e.line, e.fancyLine, maxWidth) + "  " + username
	}

	for _, line := range treeLines {
		pt.writeLine(line)
	}
}
