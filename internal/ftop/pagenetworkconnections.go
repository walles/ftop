package ftop

import (
	"github.com/walles/ftop/internal/processes"
)

// Seam for testing, see resolveAddressesViaDns()
var resolveAddresses = resolveAddressesViaDns

// The connections between this process and the rest of the world, plus the
// ports it listens on.
//
// Connections to other processes on this machine go into the Inter Process
// Communication section instead.
func (u *Ui) networkConnectionsForPaging(
	currentProcess *processes.Process,
	allProcesses []*processes.Process,
	sockets socketListing,
	pt *pageText,
) {
	const title = "Network Connections"

	pt.writeTitle(title)

	// TODO: List the listening ports and the connections to remote hosts
}

// Reverse resolves addresses into host names, mapped by address.
//
// Addresses we get no answer for are left out rather than guessed at, so expect
// a partial result and fall back to the address itself. Duplicate addresses are
// fine, they are only looked up once.
//
// All lookups share one 2 second timeout, so that is the worst a call can cost
// however many addresses it is given.
func resolveAddressesViaDns(addresses []string) map[string]string {
	// TODO: Look the addresses up concurrently under one shared deadline
	return map[string]string{}
}
