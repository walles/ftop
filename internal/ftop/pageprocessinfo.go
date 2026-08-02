package ftop

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/loginhistory"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/ui"
	"github.com/walles/ftop/internal/util"
	"github.com/walles/moor/v2/pkg/moor"
	"github.com/walles/moor/v2/twin"
)

const DISPLAY_TIME_FORMAT = "2006-01-02 Mon 15:04:05MST"

var getLoggedInUsersAt = loginhistory.GetUsersAt
var getCwdsByPid = processes.GetCwdsByPid

type pageText struct {
	// Where the page goes, one line at a time. The pager renders whatever has
	// arrived so far, so the early sections are readable while the slow ones
	// are still being composed.
	out io.Writer

	titleStyle  twin.Style
	borderStyle twin.Style
}

// Concatenates the parts and hands them over in a single write.
//
// Styled lines come in a lot of pieces, and the pager is on the other end of a
// pipe, so putting the line together first saves it a handover per piece.
//
// Write errors are ignored: the only reader is the pager, and once that one is
// gone there is nobody left to show the rest of the page to.
func (pt *pageText) write(parts ...string) {
	var line strings.Builder
	for _, part := range parts {
		line.WriteString(part)
	}

	_, _ = io.WriteString(pt.out, line.String())
}

// Appends a line feed at the end of the provided string
func (pt *pageText) writeLine(line string) {
	pt.write(line, "\n")
}

func (pt *pageText) writeTitle(title string) {
	const width = 80
	trailerLength := max(2, width-2-utf8.RuneCountInString(title))
	trailer := strings.Repeat("─", trailerLength)

	// "24 bit" is fine here, if the terminal doesn't support it, the pager will
	// just down sample it as needed.
	pt.write(
		pt.borderStyle.RenderUpdateFrom(twin.StyleDefault, twin.ColorCount24bit),
		"──",
		pt.titleStyle.RenderUpdateFrom(pt.borderStyle, twin.ColorCount24bit),
		title,
		pt.borderStyle.RenderUpdateFrom(pt.titleStyle, twin.ColorCount24bit),
		trailer,
		twin.StyleDefault.RenderUpdateFrom(pt.borderStyle, twin.ColorCount24bit),
		"\n",
		"\n",
	)
}

func (u *Ui) pageProcessInfo(proc *processes.Process) {
	if proc == nil {
		panic("proc is nil, can't page process info")
	}

	log.Infof("Paging info for %s", proc.String())
	err := u.screen.PauseAndCall(func() error {
		return u.buildAndPageProcessInfo(proc)
	})
	if err != nil {
		log.Infof("Failed to page %s info: %v", proc.String(), err)
	} else {
		log.Infof("Done paging info for %s", proc.String())
	}
}

func (u *Ui) buildAndPageProcessInfo(proc *processes.Process) error {
	// Snapshotted here rather than inside the composer below, since the main
	// loop replaces u.allProcesses once per frame.
	candidates := make([]*processes.Process, len(u.allProcesses))
	for i := range u.allProcesses {
		candidates[i] = &u.allProcesses[i]
	}

	pipeReader, pipeWriter := io.Pipe()

	go u.composeProcessInfo(proc, candidates, pipeWriter)

	return moor.PageFromStream(pipeReader, moor.Options{NoLineNumbers: true})
}

// Composes the page into pipeWriter, then closes it so the pager knows the page
// is complete.
//
// Meant to be run in a goroutine of its own, see buildAndPageProcessInfo().
//
// A crash while composing ends the page with a note about it, and asks ftop to
// shut down. The pager is unaffected either way, so the user gets to read both
// the page so far and the crash note before ftop exits.
func (u *Ui) composeProcessInfo(proc *processes.Process, candidates []*processes.Process, pipeWriter *io.PipeWriter) {
	// Registered before the recovery below so that it runs after it, leaving the
	// pipe open long enough for the crash note to make it into the page.
	defer func() {
		_ = pipeWriter.Close()
	}()

	defer func() {
		panicResult := recover()
		if panicResult == nil {
			return
		}

		// Into the page before reporting the crash, so that the note is there
		// by the time the pager gets the end of the page.
		pt := pageText{out: pipeWriter}
		pt.writeLine("")
		pt.writeLine(fmt.Sprintf("<Page composition crashed: %s>", panicResult))

		log.PanicHandler("main/process info composer", panicResult, debug.Stack())
	}()

	u.writeProcessInfo(proc, candidates, pipeWriter)
}

// Composes the process info page into out, one section at a time.
//
// candidates is the process list to look for working directory friends in, see
// processes.CwdFriends().
//
// This forks subprocesses and can take a while. It writes as it goes, so
// whatever is on the other end of out will see the early sections long before
// this returns.
func (u *Ui) writeProcessInfo(proc *processes.Process, candidates []*processes.Process, out io.Writer) {
	pt := pageText{
		out:         out,
		borderStyle: twin.StyleDefault.WithForeground(u.theme.Border()),
		titleStyle:  twin.StyleDefault.WithForeground(u.theme.BorderTitle()),
	}

	pt.writeTitle("Command Line")
	u.commandLineForPaging(proc, &pt)

	pt.writeLine("")
	pt.writeLine("")

	pt.writeTitle("Launch Hierarchy")
	u.launchHierarchyForPaging(proc, &pt)

	pt.writeLine("")
	pt.writeLine("")

	pt.writeTitle("Timings")
	age := time.Since(proc.StartTime())
	cpuTime := time.Duration(0)
	if proc.CpuTimeTotal != nil {
		cpuTime = *proc.CpuTimeTotal
	}
	percentCpu := 100.0 * float64(cpuTime) / float64(age)
	pt.writeLine(fmt.Sprintf(
		"Started %s ago at %s. It used %s CPU, or %s.",
		u.highlight(util.FormatDuration(age)),
		u.highlight(proc.StartTime().Format(DISPLAY_TIME_FORMAT)),
		u.highlight(util.FormatDuration(cpuTime)),
		u.highlight(util.FormatPercent(percentCpu)),
	))

	pt.writeLine("")
	pt.writeLine("")

	pt.writeTitle("Other Processes Launched Close To " + proc.String())
	u.closeLaunchesForPaging(proc, &pt)

	pt.writeLine("")
	pt.writeLine("")

	u.usersLoggedInWhenProcessStartedForPaging(proc, &pt)

	pt.writeLine("")
	pt.writeLine("")

	u.cwdFriendsForPaging(proc, candidates, &pt)

	pt.writeLine("")

	// End with a separator
	pt.writeTitle("")
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
			if a.Command() != b.Command() {
				return strings.Compare(strings.ToLower(a.Command()), strings.ToLower(b.Command()))
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
	split := proc.DisplayCommandLine()
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

	delta := func(p *processes.Process) time.Duration {
		if p.StartTime().Before(zero) {
			return zero.Sub(p.StartTime())
		}
		return p.StartTime().Sub(zero)
	}

	maxDelta := delta(procs[0])
	minDelta := delta(procs[0])
	for _, p := range procs {
		d := delta(p)
		if d > maxDelta {
			maxDelta = d
		}
		if d < minDelta {
			minDelta = d
		}
	}

	ramp := ui.NewColorRamp(
		float64(minDelta.Milliseconds()),
		float64(maxDelta.Milliseconds()),
		u.theme.Foreground(),
		u.theme.FadedForeground(),
	)

	for _, p := range procs {
		beforeOrAfter := "after"
		deltaT := p.StartTime().Sub(zero).Abs()
		if p.StartTime().Before(zero) {
			beforeOrAfter = "before"
		}

		deltaString := util.FormatDuration(deltaT) + " " + beforeOrAfter
		if deltaT.Milliseconds() == 0 {
			deltaString = "at the same time as"
		}

		style := twin.StyleDefault.WithForeground(ramp.AtValue(float64(deltaT.Milliseconds())))
		highlighted := twin.StyleDefault.WithForeground(u.theme.HighlightedForeground())
		pt.writeLine(fmt.Sprintf("%s%s launched %s%s%s %s",
			style.RenderUpdateFrom(twin.StyleDefault, twin.ColorCount24bit),
			p.String(),
			highlighted.RenderUpdateFrom(style, twin.ColorCount24bit),
			deltaString,
			twin.StyleDefault.RenderUpdateFrom(highlighted, twin.ColorCount24bit),
			proc.String(),
		))
	}
}

func (u *Ui) usersLoggedInWhenProcessStartedForPaging(proc *processes.Process, pt *pageText) {
	pt.writeTitle("Users logged in when " + proc.String() + " started")

	users, err := getLoggedInUsersAt(proc.StartTime())
	if err != nil {
		pt.writeLine("<Unable to inspect login history: " + err.Error() + ">")
		return
	}

	if len(users) == 0 {
		pt.writeLine("<Nobody found, either nobody was logged in or the wtmp logs have been rotated>")
		return
	}

	for _, user := range users {
		pt.writeLine(user)
	}
}

func (u *Ui) cwdFriendsForPaging(proc *processes.Process, candidates []*processes.Process, pt *pageText) {
	const title = "Others sharing this process' working directory"

	cwds, err := getCwdsByPid()
	if err != nil {
		pt.writeTitle(title)
		pt.writeLine("<Unable to list working directories: " + err.Error() + ">")
		return
	}

	cwd, found := cwds[proc.Pid]
	if !found {
		pt.writeTitle(title)
		pt.writeLine("<Working directory unknown, try again or try \"sudo ftop\">")
		return
	}

	pt.writeTitle(title + " (" + cwd + ")")

	if cwd == "/" {
		pt.writeLine("<Working directory too common, never mind>")
		return
	}

	friends := processes.CwdFriends(proc, candidates, cwds)
	if len(friends) == 0 {
		pt.writeLine("<Nobody else shares this working directory>")
		return
	}

	for _, friend := range friends {
		pt.writeLine(friend.String())
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
