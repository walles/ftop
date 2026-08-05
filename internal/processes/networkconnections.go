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

// Which way a connection's arrow points.
//
// Two different facts share the one arrow. For a socket it is which end dialed
// the other, worked out from the ports this machine listens on, so it can come
// out backwards for a connection whose listening socket we cannot see; see
// directionAndPort(). For a pipe it is which way the data flows, read off the
// access mode of the end we hold; see pipeDirection().
type Direction int

const (
	// The peer dialed us, or writes into a pipe we read
	DirectionIncoming Direction = iota

	// We dialed the peer, or write into a pipe they read
	DirectionOutgoing

	// No telling. Every UDP connection is this, UDP having no listening state
	// to compare a port against, and so is every anonymous pipe on macOS, whose
	// lsof reports no access mode for one.
	DirectionUnknown
)

// Some number of connections between one process and one peer, all of them
// speaking the same protocol to or from the same port.
type Connection struct {
	Peer      Peer
	Protocol  Protocol
	Direction Direction

	// The port being served, never the ephemeral one the client picked. The
	// peer's port when Direction is DirectionUnknown, there being no telling
	// which of the two is the served one.
	//
	// Zero for a connection carried by something that has no ports at all, a
	// pipe being one, and then rendered as the bare protocol.
	Port int

	// True for a port we accept connections on. Such a connection has no peer,
	// and its Direction is DirectionIncoming for lack of anything better.
	Listening bool

	// How many connections this line stands for, at least 1. Always 1 for
	// listening ports, which are one port however many sockets hold them open.
	Count int
}

// The TCP and UDP connections of proc, aggregated and sorted for display.
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
// say about them. That takes most UDP sockets with it, a UDP server's bound
// socket included, since UDP has no listening state to tell one from an
// ephemeral source port.
//
// Which costs UDP its peers as well: a UDP server serves every client from one
// bound socket and never connects it, so there is no reversed pair to match, and
// a process talking to a local UDP server gets that server's address for a peer
// rather than the server itself. Expect local UDP to come back as Peer.Pid 0 on
// a loopback address unless both ends happen to have connected their sockets.
//
// The result is empty if proc holds no sockets, or if it is missing from
// socketsByPid because we aren't allowed to inspect it.
//
// Which end dialed which is worked out from the ports this machine listens on,
// so a connection whose listening socket is nowhere to be seen, having been
// closed or held by a process we may not inspect, can come out backwards, and
// then reported on the client's port rather than the server's. UDP connections
// are always DirectionUnknown, see directionAndPort().
//
// The ordering is listening ports first, then incoming connections, then
// outgoing ones, then the ones we can't tell the direction of; by peer name, PID
// and port within each group.
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
			counts[Connection{
				Protocol:  socket.Protocol,
				Direction: DirectionIncoming,
				Port:      port,
				Listening: true,
			}] = 1
			continue
		}

		if socket.Remote == "" {
			// Bound, with nobody at the other end. Nothing we can say honestly:
			// see this function's own doc comment for what that costs UDP, which
			// arrives here far more often than TCP does.
			continue
		}

		if isTheFarEndOfOurOwnConnection(socket, ourPairs) {
			continue
		}

		direction, port := directionAndPort(socket, listeningEndpoints, listeningWildcardPorts)

		counts[Connection{
			Peer:      peerOf(socket, peerPids, names),
			Protocol:  socket.Protocol,
			Direction: direction,
			Port:      port,
		}]++
	}

	connections := make([]Connection, 0, len(counts))
	for connection, count := range counts {
		connection.Count = count
		connections = append(connections, connection)
	}

	SortConnections(connections)

	return connections
}

// Sorts connections into display order, in place.
//
// Exported because a page section showing more than one kind of connection has
// to merge the lists and sort the result, see compareConnections() for what the
// order is.
func SortConnections(connections []Connection) {
	slices.SortFunc(connections, compareConnections)
}

// Which end of socket dialed the other, and the port worth reporting it on.
//
// Only TCP tells us who dialed whom. Anything else comes back
// DirectionUnknown on the peer's port, there being no way to tell which of the
// two ports is the served one.
//
// listeningEndpoints and listeningWildcardPorts are the machine's listen sets,
// see listenSets(). socket is expected to have a peer; a listening socket has no
// direction to work out.
//
// A connection to a port this machine listens on is one somebody else dialed.
// The whole machine's ports rather than just our own, because the process
// holding an accepted socket often isn't the one holding the listener: an
// accept-then-fork server keeps the listener in the parent, every sshd session
// child being one, and a socket activated server never holds one at all.
//
// Known limit: a server that closes its listening socket once it has accepted,
// as "nc -l" does, leaves no listening port anywhere on the machine, so its
// connections come out backwards, as if it had dialed the client on the client's
// ephemeral port. Telling those apart would mean guessing from the port numbers,
// and ephemeral ranges are platform specific while servers do listen high. Same
// for a listener held by a process we aren't allowed to inspect, which is why
// this gets more accurate as root.
//
// The machine-wide set gets it wrong the other way around too: dialing out from
// a local port that anything on this machine listens on reads as incoming, and
// then the port reported is our own instead of the server's. That takes an
// ephemeral port colliding with a listening one, so unlike the cases above it is
// a coincidence rather than a pattern.
func directionAndPort(
	socket Socket,
	listeningEndpoints map[string]bool,
	listeningWildcardPorts map[int]bool,
) (Direction, int) {
	_, remotePort := splitEndpoint(socket.Remote)

	if socket.Protocol != ProtocolTcp {
		// Only TCP tells us who dialed whom. lsof reports no state at all for a
		// UDP socket, and a UDP socket is bound as soon as it sends, so there is
		// no listening port anywhere on the machine to compare ours against. Of
		// the two ports the peer's is the one more likely to mean something, a
		// process talking to a UDP service being the common case.
		return DirectionUnknown, remotePort
	}

	_, localPort := splitEndpoint(socket.Local)
	if listeningEndpoints[socket.Local] || listeningWildcardPorts[localPort] {
		return DirectionIncoming, localPort
	}

	return DirectionOutgoing, remotePort
}

// Who is at the other end of socket: the process holding the same connection
// with the endpoints the other way around, or the remote address if we can find
// no such process.
//
// peerPids and names are the indexes built by peerPidsByEndpointPair() and a
// PID to command name mapping, respectively.
func peerOf(socket Socket, peerPids map[string]int, names map[int]string) Peer {
	peerPid, found := peerPids[endpointPairKey(socket.Protocol, socket.Remote, socket.Local)]
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

	return ourPairs[endpointPairKey(socket.Protocol, socket.Remote, socket.Local)]
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

		pairs[endpointPairKey(socket.Protocol, socket.Local, socket.Remote)] = true
	}

	return pairs
}

// Our sockets with the repeats left out, recognized by their endpoints.
//
// One socket reaches us several times over in more than one way: some lsof
// versions report an open file once per thread of the process holding it, and a
// socket open on several file descriptors, by dup(2) or by being inherited as
// stdin, stdout and stderr, is reported once per descriptor. A connection is its
// protocol plus its four endpoint numbers, so two sockets of one process carrying
// the same five are the same socket, whichever descriptors they arrived on.
//
// Listening is part of what identifies a socket as well, so that a listener isn't
// mistaken for the bound but unconnected socket that shares its address.
func deduplicateBySocket(sockets []Socket) []Socket {
	type socketIdentity struct {
		protocol  Protocol
		local     string
		remote    string
		listening bool
	}

	seen := map[socketIdentity]bool{}

	var deduplicated []Socket
	for _, socket := range sockets {
		identity := socketIdentity{
			protocol:  socket.Protocol,
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
// for why. All of them TCP in practice, UDP having no listening state.
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

			key := endpointPairKey(socket.Protocol, socket.Local, socket.Remote)
			lowestSoFar, found := peerPids[key]
			if found && lowestSoFar < pid {
				continue
			}

			peerPids[key] = pid
		}
	}

	return peerPids
}

// How a connection is spelled in the peer index: its protocol, then one end and
// the other.
//
// Looking up a socket of ours with its endpoints reversed finds the process at
// the other end of it. A connection is its protocol plus its four endpoint
// numbers, so there is at most one such connection, held by at least one process
// if we are allowed to see it at all.
//
// The protocol belongs in the key because a TCP and a UDP connection can carry
// the very same four numbers while having nothing to do with each other.
func endpointPairKey(protocol Protocol, local string, remote string) string {
	return string(protocol) + " " + local + "->" + remote
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

// Listening ports first, then incoming connections, then outgoing ones, then the
// ones nobody can tell the direction of, so that each way of drawing a connection
// stays in one block of the listing. Protocol next, so that a block of pipes and
// a block of sockets don't interleave, their descriptions being the column a
// reader scans. Then by peer name, peer PID and port, all of which are only tie
// breakers, there to make the order the same every time.
//
// Sorting by protocol used to be unnecessary, TCP being the only protocol
// reaching the first three groups and UDP the only one reaching the last, which
// made the blocks protocol-pure for free. Pipes reach both the incoming and the
// outgoing group and ended that.
func compareConnections(a Connection, b Connection) int {
	return cmp.Or(
		cmp.Compare(sortGroup(a), sortGroup(b)),
		strings.Compare(string(a.Protocol), string(b.Protocol)),
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

	if connection.Direction == DirectionOutgoing {
		return 2
	}

	return 3
}
