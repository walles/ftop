package processes

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
)

// Who is at the other end of a connection.
type Peer struct {
	// The peer process' command name, or its address when Pid is 0. Empty when
	// we are listening, since nobody is there yet, and empty for a process we
	// have no name for.
	Name string

	// The peer process' PID, or 0 when the peer is a remote host rather than a
	// process on this machine.
	Pid int
}

// Which end dialed the other.
//
// Worked out from the ports this machine listens on, which can get it backwards
// for a connection whose listening socket we cannot see, see
// NetworkConnections().
type Direction int

const (
	// The peer dialed us
	DirectionIncoming Direction = iota

	// We dialed the peer
	DirectionOutgoing
)

// Some number of TCP connections between one process and one peer, all of them
// to or from the same port.
type Connection struct {
	Peer      Peer
	Direction Direction

	// The port being served, never the ephemeral one the client picked.
	Port int

	// True for a port we accept connections on. Such a connection has no peer,
	// and its Direction is DirectionIncoming for lack of anything better.
	Listening bool

	// How many connections this line stands for, at least 1. Always 1 for
	// listening ports, which are one port however many sockets hold them open.
	Count int
}

// The TCP connections of proc, aggregated and sorted for display.
//
// socketsByPid is every socket we could see, see GetSocketsByPid(); the sockets
// of other processes are what lets us name the process at the other end of a
// connection. allProcesses turns those PIDs into names; pass every process you
// know about. A peer we find no process for keeps its PID and gets no name.
//
// A socket inherited across a fork belongs to several processes at once, and
// then the lowest of their PIDs is the one reported as the peer.
//
// Peers that aren't processes on this machine come back as Peer.Pid 0 with the
// peer's address as Peer.Name, unresolved: this does no name lookups.
//
// Sockets that neither listen nor have a peer are left out; lsof reports those
// for sockets that are bound but were never connected, and there is nothing to
// say about them.
//
// The result is empty if proc holds no TCP sockets, or if it is missing from
// socketsByPid because we aren't allowed to inspect it.
//
// Which end dialed which is worked out from the ports this machine listens on,
// so a connection whose listening socket is nowhere to be seen, having been
// closed or held by a process we may not inspect, can come out backwards, and
// then reported on the client's port rather than the server's.
//
// The ordering is listening ports first, then incoming connections, then
// outgoing ones; by peer name, PID and port within each group.
func NetworkConnections(proc *Process, allProcesses []*Process, socketsByPid map[int][]Socket) []Connection {
	ourSockets := deduplicateBySocket(socketsByPid[proc.Pid])
	if len(ourSockets) == 0 {
		return nil
	}

	peerPids := peerPidsByEndpointPair(socketsByPid)
	listeningEndpoints, listeningWildcardPorts := listenSets(socketsByPid)
	ourPairs := endpointPairs(ourSockets)

	names := map[int]string{}
	for _, candidate := range allProcesses {
		names[candidate.Pid] = candidate.Command()
	}

	// The connections we found, mapped to how many sockets stand for each of
	// them. The keys carry no Count of their own, that is what the values are.
	counts := map[Connection]int{}

	for _, socket := range ourSockets {
		if socket.Listening {
			// One listening port however many sockets hold it open: a dual
			// stack listener has one per address family, and reporting two of
			// them, or one with a count of two, would both be lies.
			_, port := splitEndpoint(socket.Local)
			counts[Connection{Direction: DirectionIncoming, Port: port, Listening: true}] = 1
			continue
		}

		if socket.Remote == "" {
			// Bound but never connected, nothing to say about it
			continue
		}

		if isTheFarEndOfOurOwnConnection(socket, ourPairs) {
			continue
		}

		_, localPort := splitEndpoint(socket.Local)

		// A connection to a port this machine listens on is one somebody else
		// dialed. The whole machine's ports rather than just our own, because
		// the process holding an accepted socket often isn't the one holding the
		// listener: an accept-then-fork server keeps the listener in the parent,
		// every sshd session child being one, and a socket activated server
		// never holds one at all.
		//
		// Known limit: a server that closes its listening socket once it has
		// accepted, as "nc -l" does, leaves no listening port anywhere on the
		// machine, so its connections come out backwards, as if it had dialed
		// the client on the client's ephemeral port. Telling those apart would
		// mean guessing from the port numbers, and ephemeral ranges are platform
		// specific while servers do listen high. Same for a listener held by a
		// process we aren't allowed to inspect, which is why this gets more
		// accurate as root.
		//
		// The machine-wide set gets it wrong the other way around too: dialing
		// out from a local port that anything on this machine listens on reads
		// as incoming, and then the port reported is our own instead of the
		// server's. That takes an ephemeral port colliding with a listening one,
		// so unlike the cases above it is a coincidence rather than a pattern.
		direction := DirectionOutgoing
		_, port := splitEndpoint(socket.Remote)
		if listeningEndpoints[socket.Local] || listeningWildcardPorts[localPort] {
			direction = DirectionIncoming
			port = localPort
		}

		counts[Connection{
			Peer:      peerOf(socket, peerPids, names),
			Direction: direction,
			Port:      port,
		}]++
	}

	connections := make([]Connection, 0, len(counts))
	for connection, count := range counts {
		connection.Count = count
		connections = append(connections, connection)
	}

	slices.SortFunc(connections, compareConnections)

	return connections
}

// Who is at the other end of socket: the process holding the same connection
// with the endpoints the other way around, or the remote address if we can find
// no such process.
//
// peerPids and names are the indexes built by peerPidsByEndpointPair() and a
// PID to command name mapping, respectively.
func peerOf(socket Socket, peerPids map[string]int, names map[int]string) Peer {
	peerPid, found := peerPids[endpointPairKey(socket.Remote, socket.Local)]
	if found {
		// The name can be missing: lsof runs after the process listing, so the
		// peer may be a process that didn't exist yet when we listed them.
		return Peer{Name: names[peerPid], Pid: peerPid}
	}

	address, _ := splitEndpoint(socket.Remote)

	return Peer{Name: address}
}

// True if socket is one end of a connection whose other end is ours as well, and
// the other end is the one we report the connection by.
//
// ourPairs is the endpoint pairs of our own sockets, see endpointPairs().
//
// A process dialing a port of its own holds both ends of the connection, and
// that is one connection deserving one line. Which of the two sockets to keep is
// arbitrary, so it goes by whichever local endpoint sorts first, for the sake of
// reporting the same one every time.
func isTheFarEndOfOurOwnConnection(socket Socket, ourPairs map[string]bool) bool {
	if socket.Local <= socket.Remote {
		return false
	}

	return ourPairs[endpointPairKey(socket.Remote, socket.Local)]
}

// The connections held by sockets, keyed the way endpointPairKey() spells them.
//
// Sockets with nobody at the other end are left out, having no pair to speak of.
func endpointPairs(sockets []Socket) map[string]bool {
	pairs := map[string]bool{}

	for _, socket := range sockets {
		if socket.Remote == "" {
			continue
		}

		pairs[endpointPairKey(socket.Local, socket.Remote)] = true
	}

	return pairs
}

// Our sockets with the repeats left out, recognized by their endpoints.
//
// One socket reaches us several times over in more than one way: some lsof
// versions report an open file once per thread of the process holding it, and a
// socket open on several file descriptors, by dup(2) or by being inherited as
// stdin, stdout and stderr, is reported once per descriptor. A TCP connection is
// its four endpoint numbers, so two sockets of one process carrying the same four
// are the same socket, whichever descriptors they arrived on.
//
// Listening is part of what identifies a socket as well, so that a listener isn't
// mistaken for the bound but unconnected socket that shares its address.
func deduplicateBySocket(sockets []Socket) []Socket {
	type socketIdentity struct {
		local     string
		remote    string
		listening bool
	}

	seen := map[socketIdentity]bool{}

	var deduplicated []Socket
	for _, socket := range sockets {
		identity := socketIdentity{
			local:     socket.Local,
			remote:    socket.Remote,
			listening: socket.Listening,
		}

		if seen[identity] {
			continue
		}

		seen[identity] = true
		deduplicated = append(deduplicated, socket)
	}

	return deduplicated
}

// The endpoints anything on this machine accepts connections on, plus the ports
// of those listeners that accept connections to any address at all.
//
// Every process' sockets rather than one process' own, see NetworkConnections()
// for why.
//
// Sockets accepted by a wildcard listener report a concrete local address, which
// never matches the "*:8082" the listener is bound to, so for those the port is
// all there is to go by. A listener bound to one address only ever accepts
// connections to that address, and gets to be matched in full.
func listenSets(socketsByPid map[int][]Socket) (endpoints map[string]bool, wildcardPorts map[int]bool) {
	endpoints = map[string]bool{}
	wildcardPorts = map[int]bool{}

	for _, sockets := range socketsByPid {
		for _, socket := range sockets {
			if !socket.Listening {
				continue
			}

			address, port := splitEndpoint(socket.Local)
			if address == "*" {
				wildcardPorts[port] = true
				continue
			}

			endpoints[socket.Local] = true
		}
	}

	return endpoints, wildcardPorts
}

// Maps connections to the lowest numbered PID holding one, keyed by endpoint
// pair the way endpointPairKey() spells it.
//
// A socket inherited across a fork is held by parent and child alike, and then
// the lowest PID is the one we keep. Arbitrary, but stable: naming a different
// process every time the same connection is looked at would be worse.
func peerPidsByEndpointPair(socketsByPid map[int][]Socket) map[string]int {
	peerPids := map[string]int{}

	for pid, sockets := range socketsByPid {
		for _, socket := range sockets {
			if socket.Remote == "" {
				// Nobody at the other end of this one, so nobody's peer either
				continue
			}

			key := endpointPairKey(socket.Local, socket.Remote)
			lowestSoFar, found := peerPids[key]
			if found && lowestSoFar < pid {
				continue
			}

			peerPids[key] = pid
		}
	}

	return peerPids
}

// How a connection is spelled in the peer index: one end, then the other.
//
// Looking up a socket of ours with its endpoints reversed finds the process at
// the other end of it. A TCP connection is its four endpoint numbers, so there
// is at most one such connection, held by at least one process if we are allowed
// to see it at all.
func endpointPairKey(local string, remote string) string {
	return local + "->" + remote
}

// The address and port of "127.0.0.1:8080", "*:7000" or "[::1]:8081", with the
// IPv6 brackets off: they are there to keep the address apart from the port, and
// the address on its own needs none.
//
// The port is 0 for an endpoint carrying none, like the "*:*" lsof reports for a
// socket that was created but never bound. Those sockets are dropped rather than
// reported on a port of 0, see NetworkConnections().
func splitEndpoint(endpoint string) (address string, port int) {
	colon := strings.LastIndex(endpoint, ":")
	if colon == -1 {
		return endpoint, 0
	}

	address = strings.Trim(endpoint[:colon], "[]")

	port, err := strconv.Atoi(endpoint[colon+1:])
	if err != nil {
		return address, 0
	}

	return address, port
}

// Listening ports first, then incoming connections, then outgoing ones, so that
// each way of drawing a connection stays in one block of the listing. Then by
// peer name, peer PID and port, all of which are only tie breakers, there to
// make the order the same every time.
func compareConnections(a Connection, b Connection) int {
	return cmp.Or(
		cmp.Compare(sortGroup(a), sortGroup(b)),
		strings.Compare(a.Peer.Name, b.Peer.Name),
		cmp.Compare(a.Peer.Pid, b.Peer.Pid),
		cmp.Compare(a.Port, b.Port),
	)
}

// Which block of the listing a connection belongs in, see compareConnections().
func sortGroup(connection Connection) int {
	if connection.Listening {
		return 0
	}

	if connection.Direction == DirectionIncoming {
		return 1
	}

	return 2
}
