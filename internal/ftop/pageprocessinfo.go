package ftop

import (
	"fmt"
	"io"
	"runtime/debug"
	"sync"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/moor/v2/pkg/moor"
	"github.com/walles/moor/v2/twin"
)

const DISPLAY_TIME_FORMAT = "2006-01-02 Mon 15:04:05MST"

// Seams for testing. Every section that collects by forking a subprocess declares
// its collector this way, so that a page test can hand it a listing and assert the
// rendered result without an lsof on the machine having to contain one.
var getSocketsByPid = processes.GetSocketsByPid
var getPipeEndsByPid = processes.GetPipeEndsByPid
var getUnixSocketsByPid = processes.GetUnixSocketsByPid

// The TCP and UDP sockets of every process we were allowed to inspect, or the
// error that came of trying to list them.
//
// Both connection sections render from one listing, so that they can't disagree
// about a connection that came or went between two lsof runs. They still render
// their own error states, which is what the error is doing here.
type socketListing struct {
	byPid map[int][]processes.Socket
	err   error
}

// The pipe ends of every process we were allowed to inspect, or the error that
// came of trying to list them.
//
// A listing of its own rather than part of socketListing: it takes a second lsof
// invocation, an unfiltered and slower one, and only the Inter Process
// Communication section has any use for it.
type pipeListing struct {
	byPid map[int][]processes.PipeEnd
	err   error
}

// The unix domain sockets of every process we were allowed to inspect, or the
// error that came of trying to list them.
//
// A listing of its own rather than part of the pipe one, though both of them
// feed the Inter Process Communication section and nothing else. "-U" selects
// unix sockets, where pipes have no filter flag at all and cost an unfiltered
// listing of every open file on the machine, so riding along would make this the
// slower of two jobs it doesn't need — and would take both kinds down whenever
// either lsof failed.
type unixSocketListing struct {
	byPid map[int][]processes.UnixSocket
	err   error
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
// allProcesses is the process list to look for working directory friends and
// connection peers in, see processes.CwdFriends() and
// processes.NetworkConnections().
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

	// Listed once for both sections that need it, and not before, so that the
	// sections above it are on their way to the pager while lsof runs.
	sockets := sync.OnceValue(func() socketListing {
		byPid, err := getSocketsByPid()
		return socketListing{byPid: byPid, err: err}
	})

	// Listed lazily as well, and separately: this is the unfiltered lsof, which
	// costs about twice what the socket one does, and only one section wants it.
	pipes := sync.OnceValue(func() pipeListing {
		byPid, err := getPipeEndsByPid()
		return pipeListing{byPid: byPid, err: err}
	})

	// Filtered by "-U", so cheap the way the socket listing is, and lazy for the
	// same reason both the others are.
	unixSockets := sync.OnceValue(func() unixSocketListing {
		byPid, err := getUnixSocketsByPid()
		return unixSocketListing{byPid: byPid, err: err}
	})

	sections := []func(){
		func() { u.commandLineForPaging(proc, &pt) },
		func() { u.launchHierarchyForPaging(proc, &pt) },
		func() { u.timingsForPaging(proc, &pt) },
		func() { u.closeLaunchesForPaging(proc, &pt) },
		func() { u.usersLoggedInWhenProcessStartedForPaging(proc, &pt) },
		func() { u.cwdFriendsForPaging(proc, allProcesses, &pt) },
		func() { u.ipcConnectionsForPaging(proc, allProcesses, sockets(), pipes(), unixSockets(), &pt) },
		func() { u.networkConnectionsForPaging(proc, allProcesses, sockets(), &pt) },
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
