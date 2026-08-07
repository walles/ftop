package processes

import (
	"errors"
)

// One unix domain socket held open by some process, as reported by lsof.
//
// Which fields are populated says what the socket is: a client's connected
// socket names its peer, a listener and the sockets accepted on it carry the
// path they are bound to, and no socket carries both. Which makes the naming
// one directional, unlike a pipe's mutual pair, and that is what
// UnixSocketConnections() reads a connection's direction off.
type UnixSocket struct {
	// lsof's file descriptor number, "3" or similar. Not an identity: one socket
	// is reported once per descriptor it is open on, and the Device is what
	// identifies it instead.
	Fd string

	// lsof's lowercase "d" column, the kernel address of this very socket,
	// "0xd82e0ac85b8e6a97". A string rather than a number because it is compared
	// and never counted with.
	Device string

	// The Device of the socket at the other end, from lsof's "n->0x..." name.
	// Empty for a socket lsof doesn't name that way, which is every socket
	// carrying a Path, and empty for one whose peer is gone.
	PeerDevice string

	// The file system path this socket is bound to, "/tmp/probe.sock". Carried by
	// a listener and by every socket accepted on it, empty for a socket that
	// dialed one of those and empty for both ends of a socketpair(2).
	Path string
}

// Maps PIDs to the unix domain sockets held open by the corresponding
// processes.
//
// The listing is partial: processes we aren't allowed to inspect are missing
// from the map, so expect only a fraction of the running processes when not
// running as root. Processes without any unix sockets are missing as well.
//
// This forks lsof, which takes a fraction of a second. Too slow for calling
// once per frame, fine for on-demand lookups.
func GetUnixSocketsByPid() (map[int][]UnixSocket, error) {
	// FIXME: Stub, so that the tests can be reviewed before this is written.
	return nil, errors.New("not implemented")
}

// Parses the output of "lsof -n -w -U -F pfnd0", which comes in NUL terminated
// fields, one line per process and then one line per socket:
//
//	p78879\0
//	f3\0d0xe396ab59862314e8\0n/tmp/probe.sock\0
//	f4\0d0xd82e0ac85b8e6a97\0n/tmp/probe.sock\0
//	f0\0d0x520254f296b152e3\0n->(none)\0
//	p78881\0
//	f3\0d0x628a5982efaac095\0n->0xd82e0ac85b8e6a97\0
//
// A server and its client. The server's first two sockets are the listener and
// the socket it accepted the client on, both named by the path they are bound
// to, and its third has lost whoever was at the other end. The client names the
// accepted socket by that socket's device and carries no path.
//
// Those are the only three shapes a name comes in, and a record has a peer or a
// path and never both: measured on a quiet macOS laptop, 551 records, 462 with a
// peer, 82 with a path, none with the two of them.
//
// "-U" selects unix sockets and nothing else, so unlike lsofPipeParser this
// needs no type field to tell its own records apart from the rest.
type lsofUnixSocketParser struct {
	unixSocketsByPid map[int][]UnixSocket

	// PID from the most recent "p" field, or -1 before the first one
	pid int
}

func newLsofUnixSocketParser() lsofUnixSocketParser {
	return lsofUnixSocketParser{
		unixSocketsByPid: map[int][]UnixSocket{},
		pid:              -1,
	}
}

func (parser *lsofUnixSocketParser) parseLine(line string) error {
	// FIXME: Stub, so that the tests can be reviewed before this is written.
	_ = line

	return nil
}
