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
	pipes pipeListing,
	pt *pageText,
) {
	const title = "Inter Process Communication"

	pt.writeTitle(title)

	// Two lsof invocations, so either one can fail while the other returns
	// something worth showing.
	if sockets.err != nil {
		pt.writeLine("<Unable to list sockets: " + sockets.err.Error() + ">")
	}

	if pipes.err != nil {
		pt.writeLine("<Unable to list pipes: " + pipes.err.Error() + ">")
	}

	if sockets.err != nil && pipes.err != nil {
		// Having looked nowhere, "no connections found" below would be a lie, so
		// the errors are all there is to say.
		return
	}

	// A failed listing arrives as a nil map, which both of these have nothing to
	// say about, so neither call needs guarding.
	var ipcConnections []processes.Connection
	for _, connection := range processes.NetworkConnections(currentProcess, allProcesses, sockets.byPid) {
		if connection.Peer.Pid == 0 {
			// A remote host, or a port we listen on. Either way the Network
			// Connections section is where it belongs.
			continue
		}

		ipcConnections = append(ipcConnections, connection)
	}

	// No such partitioning for pipes: a pipe is between processes by nature, so
	// every one of these belongs here.
	pipeConnections := processes.PipeConnections(currentProcess, allProcesses, pipes.byPid)
	ipcConnections = append(ipcConnections, pipeConnections...)

	// Two sorted lists appended is not a sorted list, and the sections' blocks
	// are what the ordering is for.
	processes.SortConnections(ipcConnections)

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
