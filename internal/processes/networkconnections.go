package processes

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
// The ordering is listening ports first, then incoming connections, then
// outgoing ones; by peer name, PID and port within each group.
func NetworkConnections(proc *Process, allProcesses []*Process, socketsByPid map[int][]Socket) []Connection {
	// TODO: Match up peers, work out directions, aggregate and sort
	return nil
}
