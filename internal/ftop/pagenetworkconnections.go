package ftop

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/walles/ftop/internal/log"
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

	if sockets.err != nil {
		pt.writeLine("<Unable to list sockets: " + sockets.err.Error() + ">")
		return
	}

	var networkConnections []processes.Connection
	var addresses []string
	for _, connection := range processes.NetworkConnections(currentProcess, allProcesses, sockets.byPid) {
		if connection.Peer.Pid != 0 {
			// A process on this machine, which is what the Inter Process
			// Communication section is for
			continue
		}

		networkConnections = append(networkConnections, connection)

		if connection.Peer.Name != "" {
			// A remote host, named by its address until we resolve it. Listening
			// ports have no peer and nothing to resolve.
			addresses = append(addresses, connection.Peer.Name)
		}
	}

	if len(networkConnections) == 0 {
		pt.writeLine("<No connections found>")
		return
	}

	names := resolveAddresses(addresses)

	u.writeConnectionLines(currentProcess, networkConnections, func(peer processes.Peer) string {
		name, found := names[peer.Name]
		if !found {
			// Unresolvable, and its address is a correct answer anyway
			return peer.Name
		}

		return name
	}, pt)
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
	unique := map[string]bool{}
	for _, address := range addresses {
		unique[address] = true
	}

	// Shared by all the lookups, so that what this call costs is a number we
	// pick rather than one that grows with the number of peers.
	//
	// On macOS Go resolves through the system resolver, which means the deadline
	// hands control back to us on time while the lookup itself keeps running in
	// its goroutine. Harmless, but it can look like a leak in a trace.
	deadline, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var resolver net.Resolver
	var lock sync.Mutex
	var lookups sync.WaitGroup

	names := map[string]string{}
	unresolved := 0

	for address := range unique {
		lookups.Go(func() {
			hostnames, err := resolver.LookupAddr(deadline, address)

			lock.Lock()
			defer lock.Unlock()

			if err != nil || len(hostnames) == 0 {
				unresolved++
				return
			}

			// Reverse lookups come back fully qualified, with the root domain's
			// trailing dot and all
			names[address] = strings.TrimSuffix(hostnames[0], ".")
		})
	}

	lookups.Wait()

	if unresolved > 0 {
		// Once with a count rather than once per address: a machine with 200
		// unresolvable peers would otherwise get 200 near identical log lines.
		log.Debugf("Reverse DNS found no name for %d of %d addresses", unresolved, len(unique))
	}

	return names
}
