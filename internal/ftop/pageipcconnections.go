package ftop

import (
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

	// TODO: List the connections whose peer is a process
}
