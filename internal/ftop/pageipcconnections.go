package ftop

import (
	"fmt"

	"github.com/walles/ftop/internal/processes"
)

// The connections between this process and other processes on this machine.
//
// Connections to remote hosts, and the ports we listen on, go into the Network
// Connections section instead.
func (u *Ui) ipcConnectionsForPaging(
	currentProcess *processes.Process,
	allProcesses []*processes.Process,
	sockets socketListing,
	pt *pageText,
) {
	const title = "Inter Process Communication"

	pt.writeTitle(title)

	if sockets.err != nil {
		// Without a listing there is nothing for the caveat below to be a caveat
		// about, so the error is all there is to say.
		pt.writeLine("<Unable to list sockets: " + sockets.err.Error() + ">")
		return
	}

	// Above the connections rather than below them: a caveat that changes how a
	// list is read has to arrive before the list, and this page goes into a pager
	// where the reader may never reach the bottom. Grow this as more kinds of IPC
	// land, and delete it once nothing is missing.
	pt.writeLine("<Detected: TCP. Not detected: UDP, pipes, unix sockets>")

	var ipcConnections []processes.Connection
	for _, connection := range processes.NetworkConnections(currentProcess, allProcesses, sockets.byPid) {
		if connection.Peer.Pid == 0 {
			// A remote host, or a port we listen on. Either way the Network
			// Connections section is where it belongs.
			continue
		}

		ipcConnections = append(ipcConnections, connection)
	}

	if len(ipcConnections) == 0 {
		pt.writeLine("<No connections found>")
		return
	}

	u.writeConnectionLines(currentProcess, ipcConnections, processPeerLabel, pt)
}

// What to call a peer process: its command name and PID, or its PID alone when
// we have no name for it.
//
// A peer without a name is a process that didn't exist yet when we listed the
// processes, since lsof runs after that.
func processPeerLabel(peer processes.Peer) string {
	if peer.Name == "" {
		return fmt.Sprintf("PID %d", peer.Pid)
	}

	return fmt.Sprintf("%s(%d)", peer.Name, peer.Pid)
}
