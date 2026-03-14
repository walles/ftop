package ftop

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/util"
	"github.com/walles/moor/v2/pkg/moor"
	"github.com/walles/moor/v2/twin"
)

type pageText struct {
	text        strings.Builder
	headerStyle twin.Style
}

// Appends a line feed at the end of the provided string
func (pt *pageText) writeLine(line string) {
	// pt.text.WriteString("  ")
	pt.text.WriteString(line)
	pt.text.WriteRune('\n')
}

func (pt *pageText) writeHeader(header string) {
	notColored := twin.StyleDefault

	// "24 bit" is fine here, if the terminal doesn't support it, the pager will
	// just down sample it as needed.
	prefix := pt.headerStyle.RenderUpdateFrom(notColored, twin.ColorCount24bit)
	suffix := notColored.RenderUpdateFrom(pt.headerStyle, twin.ColorCount24bit)
	pt.text.WriteString(prefix + "-- " + header + " --" + suffix + "\n\n")
}

func (pt *pageText) String() string {
	return pt.text.String()
}

func (u *Ui) pageProcessInfo(proc *processes.Process) error {
	if proc == nil {
		panic("proc is nil, can't page process info")
	}

	pt := pageText{
		headerStyle: twin.StyleDefault.WithForeground(u.theme.Border()),
	}

	pt.writeHeader("Command Line")
	u.commandLineForPaging(proc, &pt)

	pt.writeLine("")
	pt.writeLine("")

	pt.writeHeader("Launch Hierarchy")
	u.launchHierarchyForPaging(proc, &pt)

	pt.writeLine("")
	pt.writeLine("")

	pt.writeHeader("Timings")
	age := time.Since(proc.StartTime())
	cpuTime := proc.CpuTimeOrZero()
	percentCpu := 100.0 * float64(cpuTime) / float64(age)
	pt.writeLine(fmt.Sprintf(
		"Started %s ago at %s. It used %s CPU, or %s.",
		u.highlight(util.FormatDuration(age)),
		u.highlight(proc.StartTime().Format("2006-01-02 15:04:05")),
		u.highlight(util.FormatDuration(cpuTime)),
		u.highlight(util.FormatPercent(percentCpu)),
	))

	pt.writeLine("")
	pt.writeLine("")

	pt.writeHeader("Other Processes Launched Close To " + proc.String())
	u.closeLaunchesForPaging(proc, &pt)

	return moor.PageFromString(pt.String(), moor.Options{NoLineNumbers: true})
}

func (u *Ui) launchHierarchyForPaging(proc *processes.Process, pt *pageText) {
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
			entries = append(entries, entry{line, line, child})
			maxWidth = max(maxWidth, utf8.RuneCountInString(line))
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
		padding := strings.Repeat(" ", maxWidth-utf8.RuneCountInString(e.line))

		username := e.process.Username
		switch username {
		case currentUsername:
			// This block intentionally left blank
		case "root":
			username = dimPrefix + username + dimSuffix
		default:
			username = boldPrefix + username + boldSuffix
		}

		treeLines[i] = e.fancyLine + padding + "  " + username
	}

	for _, line := range treeLines {
		pt.writeLine(line)
	}
}

func (u *Ui) commandLineForPaging(proc *processes.Process, pt *pageText) {
	split := proc.CommandLine()
	if len(split) == 0 {
		panic(fmt.Sprintf("process has no command line: %s", proc.String()))
	}

	binary := split[0]
	lastSlashIndex := strings.LastIndex(binary, string(os.PathSeparator))
	var firstLine string
	if lastSlashIndex == -1 {
		firstLine = u.highlight(binary)
	} else {
		path := binary[0 : lastSlashIndex+1] // "/bin/"
		command := binary[lastSlashIndex+1:] // "ls"
		firstLine = path + u.highlight(command)
	}

	pt.writeLine(firstLine)

	noMoreOptions := false
	lastWasOption := false
	for _, arg := range split[1:] {
		if arg == "--" {
			noMoreOptions = true
		}

		extraIndent := ""
		if lastWasOption {
			extraIndent = "  "
		}
		if strings.HasPrefix(arg, "-") {
			lastWasOption = true
			extraIndent = ""
		} else {
			lastWasOption = false
		}
		if noMoreOptions {
			extraIndent = ""
		}

		pt.writeLine("  " + extraIndent + arg)
	}
}

func (u *Ui) closeLaunchesForPaging(proc *processes.Process, pt *pageText) {
	procs := findCloseLaunches(proc)

	zero := proc.StartTime()
	for _, p := range procs {
		beforeOrAfter := "after"
		deltaT := p.StartTime().Sub(zero)
		if p.StartTime().Before(zero) {
			beforeOrAfter = "before"
			deltaT = -deltaT
		}

		deltaString := util.FormatDuration(deltaT) + " " + beforeOrAfter
		if deltaT.Milliseconds() == 0 {
			deltaString = "at the same time as"
		}

		pt.writeLine(fmt.Sprintf("%s was launched %s %s",
			u.highlight(p.String()),
			u.highlight(deltaString),
			proc.String(),
		))
	}
}

// Return the top closest launches, ordered by closeness, excluding the process
// itself. The answer will be 5-7 processes long.
//
// Always includes at least one process before and one after.
func findCloseLaunches(proc *processes.Process) []*processes.Process {
	allOtherProcs := getAllOtherProcesses(proc)

	// Sort by launch time closeness
	zero := proc.StartTime()
	slices.SortFunc(allOtherProcs, func(a, b *processes.Process) int {
		diffA := a.StartTime().Sub(zero).Milliseconds()
		if diffA < 0 {
			diffA = -diffA
		}

		diffB := b.StartTime().Sub(zero).Milliseconds()
		if diffB < 0 {
			diffB = -diffB
		}

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
	var before, after *processes.Process
	for _, candidate := range allOtherProcs {
		if candidate.StartTime().Before(zero) {
			before = candidate
		} else if candidate.StartTime().After(zero) && after == nil {
			after = candidate
			break
		}
	}
	if before != nil && !slices.ContainsFunc(topList, func(p *processes.Process) bool { return p.SameAs(before) }) {
		topList = append(topList, before)
	}
	if after != nil && !slices.ContainsFunc(topList, func(p *processes.Process) bool { return p.SameAs(after) }) {
		topList = append(topList, after)
	}

	sortByStartTime(topList)

	return topList
}

func sortByStartTime(procs []*processes.Process) {
	slices.SortFunc(procs, func(a, b *processes.Process) int {
		if a.StartTime().Before(b.StartTime()) {
			return -1
		}
		if a.StartTime().After(b.StartTime()) {
			return 1
		}
		return 0
	})
}

// List all other processes in the same tree in no particular order
func getAllOtherProcesses(proc *processes.Process) []*processes.Process {
	// Find the root process
	init := proc
	for init.Parent() != nil {
		init = init.Parent()
	}

	// Flatten the process tree
	allProcs := []*processes.Process{}
	var flatten func(p *processes.Process)
	flatten = func(p *processes.Process) {
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

func (u *Ui) highlight(s string) string {
	colored := twin.StyleDefault.WithForeground(u.theme.HighlightedForeground())
	notColored := twin.StyleDefault

	// "24 bit" is fine here, if the terminal doesn't support it, the pager will
	// just down sample it as needed.
	prefix := colored.RenderUpdateFrom(notColored, twin.ColorCount24bit)
	suffix := notColored.RenderUpdateFrom(colored, twin.ColorCount24bit)
	return prefix + s + suffix
}
