package processes

import "cmp"

// The unix domain socket connections of proc, aggregated and sorted for display.
//
// unixSocketsByPid is every unix socket we could see, see GetUnixSocketsByPid();
// the sockets held by other processes are what lets us name the process at the
// other end of a connection. allProcesses turns those PIDs into names; pass every
// process you know about. A peer we find no process for keeps its PID and gets no
// name.
//
// Sockets we find no peer for are left out. The socket at the other end may be
// held by a process we aren't allowed to inspect, 106 of 489 named peers on a
// quiet non-root laptop being invisible that way, and a socket whose peer is gone
// looks no different. A listening socket nobody has dialed is left out for the
// same reason: it has no peer, so it is no connection.
//
// Every connection comes back with Protocol ProtocolUnix and Port 0, a unix
// socket having no port, and with the Path it was made over where there is one.
// A socketpair(2) was made over no path and leaves it empty.
//
// A socket inherited across a fork is held by several processes at once, and then
// the lowest of their PIDs is the one reported as the peer, the way it is for an
// inherited TCP socket.
//
// Count is how many connections a line stands for. Several connections to one
// peer over one path are one line counting them, while two paths to one peer are
// a line each: two paths are two services of that peer.
func UnixSocketConnections(proc *Process, allProcesses []*Process, unixSocketsByPid map[int][]UnixSocket) []Connection {
	ourSockets := unixSocketsByPid[proc.Pid]
	if len(ourSockets) == 0 {
		return nil
	}

	socketsByDevice := unixSocketsByDevice(unixSocketsByPid)

	ourSocketsByDevice := map[string]UnixSocket{}
	for _, socket := range ourSockets {
		if socket.Device == "" {
			// Nothing identifies this one, so nothing can name it either. Indexing
			// it under the empty string would make it the peer of every socket
			// lsof named no peer for.
			continue
		}

		ourSocketsByDevice[socket.Device] = socket
	}

	names := map[int]string{}
	for _, candidate := range allProcesses {
		names[candidate.Pid] = candidate.Command()
	}

	// One entry per connection we hold an end of, keyed so that the two ways of
	// finding one — us naming their socket and them naming ours — meet in the same
	// entry instead of making two lines of it.
	matches := map[string]unixSocketMatch{}

	// The connections a socket of ours names the other end of, which is an index
	// lookup each. On macOS that is the ones we dialed and on Linux it is all of
	// them, the netlink peer edge being symmetric — either way the direction comes
	// from the paths rather than from which loop found it.
	for _, ourSocket := range ourSockets {
		if ourSocket.PeerDevice == "" {
			// Nothing at the other end that we can name: a listener, a socket whose
			// peer is gone, a peer we aren't allowed to see, or on macOS one lsof
			// named by its path rather than by a peer.
			continue
		}

		theirs, found := socketsByDevice[ourSocket.PeerDevice]
		if !found {
			// The socket at the other end is held by processes we aren't allowed
			// to inspect. Nothing to name it by.
			continue
		}

		noteUnixSocketMatch(matches, ourSocket, theirs)
	}

	// The connections named the other way around, which no socket of ours points
	// at: finding those means scanning the whole listing for a socket naming one of
	// ours. On macOS that is the connections somebody else dialed, and on Linux it
	// is the same set the loop above found. A socket naming nobody names none of
	// ours either, an empty peer being a device nothing is indexed under.
	for _, theirs := range socketsByDevice {
		ourSocket, isOurs := ourSocketsByDevice[theirs.socket.PeerDevice]
		if !isOurs {
			continue
		}

		noteUnixSocketMatch(matches, ourSocket, theirs)
	}

	// How many connections each line stands for. Two connections to one peer over
	// one path are one line counting two; over two paths they are two lines.
	counts := map[Connection]int{}
	for _, match := range matches {
		counts[Connection{
			Peer:      Peer{Name: names[match.peerPid], Pid: match.peerPid},
			Protocol:  ProtocolUnix,
			Direction: unixSocketDirection(match),

			// At most one end of a connection carries the path, that being the
			// listening socket or one accepted on it, so whichever end has one has
			// the connection's.
			Path: cmp.Or(match.ourPath, match.theirPath),
		}]++
	}

	var connections []Connection
	for connection, count := range counts {
		connection.Count = count
		connections = append(connections, connection)
	}

	SortConnections(connections)

	return connections
}

// One unix socket connection we hold an end of, as far as the matching has got
// with it.
//
// The two paths are what the direction is read off, so they are kept apart rather
// than collapsed into the one path a connection renders with; see
// unixSocketDirection().
type unixSocketMatch struct {
	// The lowest numbered PID holding the socket at the other end
	peerPid int

	// The path our end of the connection is bound to, empty for the end that
	// dialed and for a socketpair(2)
	ourPath string

	// The same for the socket at the other end
	theirPath string
}

// A unix socket together with the lowest numbered PID holding it, which is who a
// connection over it is reported as being with.
type unixSocketHolder struct {
	socket UnixSocket

	pid int
}

// Notes the connection between our socket and theirs in matches, merging it with
// whatever the same connection was already found to be.
//
// One connection can be noted several times over: from either end where both ends
// name each other, which is every connected pair on Linux, and once per end where
// both ends are ours. Every field here has to come out the same whichever order
// those arrive in, map iteration order being what decides it.
func noteUnixSocketMatch(
	matches map[string]unixSocketMatch,
	ours UnixSocket,
	theirs unixSocketHolder,
) {
	identity := unixSocketConnectionIdentity(ours, theirs.socket)

	match := matches[identity]

	// Lowest PID wins here as well as in unixSocketsByDevice(), the two ends of a
	// connection we hold both of having a holder each to be named by.
	if match.peerPid == 0 || theirs.pid < match.peerPid {
		match.peerPid = theirs.pid
	}

	// Both paths kept, and both filled in for a connection whose two ends are
	// ours: noting that one from either end swaps which socket is "ours", so this
	// is where the direction of a self connection becomes unknown.
	match.ourPath = cmp.Or(match.ourPath, ours.Path)
	match.theirPath = cmp.Or(match.theirPath, theirs.socket.Path)

	matches[identity] = match
}

// Which end of a connection dialed the other: the end without a path did.
//
// That is the whole of the rule, and unlike the socket one in directionAndPort()
// it needs nothing but the two sockets themselves — no listening ports to compare
// against, and no access modes of the kind pipeDirection() reads. There would be
// nothing to read there anyway: lsof reports the mode "u" for every unix socket
// record and says nothing by it.
//
// A path is what a socket is dialed by, so a listener and every socket accepted
// on one carry it while whoever dialed carries nothing. Both platforms agree,
// which is why this needs no GOOS switch — and it is the only fact about a unix
// socket that they do agree on, macOS having the client name its peer where a
// netlink peer edge makes both ends of every Linux pair name each other.
//
// The evidence on macOS, measured on a quiet laptop, non-root, on a later run
// than the one lsofUnixSocketParser cites: of 577 unix sockets, 489 named a peer
// and 383 of those named one we could see. All 383 came in one of exactly two
// shapes — 354 records in 164 mutual pairs, none of them carrying a path, which
// is socketpair(2), and 29 naming a socket that does carry a path and never gets
// named back, which is a client naming its server. Nothing in between. On Linux,
// verified against two real pairs and a socketpair(2) in a container: the client's
// end came back from the netlink dump with no name while the accepted end carried
// the path, and neither end of the socketpair(2) had one.
//
// DirectionUnknown when the two ends have a path each or neither has one, which
// happens two ways and neither leaves an arrow to draw. Neither end having one is
// a socketpair(2), dialed by nobody, its two ends coming into being connected.
// Both ends having one is a process that dialed a socket of its own: noting that
// connection from either end fills both paths in with the accepted end's, which
// makes us the dialer and the dialed at once.
//
// Never a backwards arrow, whatever we cannot see: both ends have to be in the
// listing for there to be a match at all, so a connection with an invisible end
// gets no line rather than a guessed direction. directionAndPort() has no such
// guarantee, inferring from a listen set that may be missing the listener.
func unixSocketDirection(match unixSocketMatch) Direction {
	weDialed := match.ourPath == ""
	theyDialed := match.theirPath == ""

	if weDialed == theyDialed {
		// Both of us, or neither, and no arrow to draw either way
		return DirectionUnknown
	}

	if weDialed {
		return DirectionOutgoing
	}

	return DirectionIncoming
}

// What identifies the connection between two unix sockets, in a form that spells
// it the same way from either end.
//
// The two kernel addresses, sorted. A connection is its two sockets, so this
// collapses every way of arriving at the same one into a single entry: both ends
// naming each other finds it from either side, and both ends being ours finds it
// once per end. Either way it is one connection and deserves one line.
func unixSocketConnectionIdentity(ours UnixSocket, theirs UnixSocket) string {
	return min(ours.Device, theirs.Device) + "\x00" + max(ours.Device, theirs.Device)
}

// Sockets lsof reported no kernel address for are left out, having nothing to be
// indexed by: keyed under the empty string they would collapse into one entry and
// stand in as the peer of one another.
//
// A socket inherited across a fork is held by parent and child alike, and then
// the lowest PID is the one kept: arbitrary, but the same answer every time,
// matching what peerPidsByEndpointPair() does for an inherited TCP socket. Two
// of 577 sockets on a quiet macOS laptop were held by more than one process, and
// one of each kind — a listener held by 5 processes, and a client's connected
// socket held by 2 — so both matching directions need the rule.
//
// This is also where a socket open on several file descriptors, by dup(2) or by
// being inherited as more than one of the standard three, stops being several
// sockets: the descriptor is no part of what identifies one. Everything else a
// socket has to say — the path it is bound to, and the peer it names — belongs to
// the socket rather than to its holder, so its several holders report it
// identically and which of them this keeps matters for the PID alone.
func unixSocketsByDevice(unixSocketsByPid map[int][]UnixSocket) map[string]unixSocketHolder {
	byDevice := map[string]unixSocketHolder{}

	for pid, sockets := range unixSocketsByPid {
		for _, socket := range sockets {
			if socket.Device == "" {
				continue
			}

			lowestSoFar, found := byDevice[socket.Device]
			if found && lowestSoFar.pid <= pid {
				continue
			}

			byDevice[socket.Device] = unixSocketHolder{socket: socket, pid: pid}
		}
	}

	return byDevice
}
