package ftop

import (
	"fmt"
	"io"
	"runtime/debug"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/moor/v2/pkg/moor"
	"github.com/walles/moor/v2/twin"
)

const DISPLAY_TIME_FORMAT = "2006-01-02 Mon 15:04:05MST"

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

// Composes the page into pipeWriter, then closes it.
//
// Meant to be run in a goroutine of its own, see buildAndPageProcessInfo().
//
// A crash while composing ends the page with a note about it, and asks ftop to
// shut down. The pager is unaffected either way, so the user gets to read both
// the page so far and the crash note before ftop exits.
func (u *Ui) composeProcessInfo(
	proc *processes.Process,
	allProcesses []*processes.Process,
	pipeWriter *io.PipeWriter,
) {
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

	u.writeProcessInfo(proc, allProcesses, pipeWriter)
}

// Composes the process info page into out, one section at a time.
//
// allProcesses is the process list to look for working directory friends in,
// see processes.CwdFriends().
//
// This forks subprocesses and can take a while. It writes as it goes, so
// whatever is on the other end of out will see the early sections long before
// this returns.
func (u *Ui) writeProcessInfo(proc *processes.Process, allProcesses []*processes.Process, out io.Writer) {
	pt := pageText{
		out:         out,
		borderStyle: twin.StyleDefault.WithForeground(u.theme.Border()),
		titleStyle:  twin.StyleDefault.WithForeground(u.theme.BorderTitle()),
	}

	sections := []func(){
		func() { u.commandLineForPaging(proc, &pt) },
		func() { u.launchHierarchyForPaging(proc, &pt) },
		func() { u.timingsForPaging(proc, &pt) },
		func() { u.closeLaunchesForPaging(proc, &pt) },
		func() { u.usersLoggedInWhenProcessStartedForPaging(proc, &pt) },
		func() { u.cwdFriendsForPaging(proc, allProcesses, &pt) },
		func() { u.ipcConnectionsForPaging(proc, &pt) },
	}

	for i, section := range sections {
		if i > 0 {
			// Sections write their own title and then their lines, so the blank
			// space between them goes here.
			pt.writeLine("")
			pt.writeLine("")
		}

		section()
	}

	// End with a separator
	pt.writeLine("")
	pt.writeTitle("")
}
