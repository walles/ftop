package processes

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/util"
)

// One unix domain socket held open by some process.
//
// Everything a macOS socket has comes from lsof. On Linux lsof reports the
// process, the descriptor, the device and the inode, and a sock_diag netlink dump
// supplies the peer and the path on top of that; see fillInPeersAndPaths().
//
// Which fields are populated says most of what the socket is: a socket bound to
// a path carries it, which is a listener and every socket accepted on one, and
// the socket that dialed such a path usually carries none. So the end without a
// path is the end that dialed, and that is what UnixSocketConnections() reads a
// connection's direction off. Neither end of a socketpair(2) has a path, which is
// exactly the case it can draw no arrow for.
//
// Usually rather than always, because a client is free to bind an address of its
// own before dialing, and then both ends carry one. Listening is what tells those
// two apart on Linux.
//
// Which end names which says nothing about direction, the two platforms
// disagreeing about it: a macOS client names the socket it dialed and nothing
// names it back, while the netlink peer edge is symmetric for stream and
// seqpacket sockets, both ends of such a Linux pair naming each other. A
// connected datagram socket is the exception at either end of that — it names its
// server and the server names nobody back, the way macOS has it.
type UnixSocket struct {
	// lsof's file descriptor number, "3" or similar. Not an identity: one socket
	// is reported once per descriptor it is open on, and unixSocketIdentity() is
	// what identifies it instead.
	Fd string

	// lsof's lowercase "d" column, the kernel address of this very socket,
	// "0xd82e0ac85b8e6a97". A string rather than a number because it is compared
	// and never counted with.
	//
	// What identifies a socket on macOS. On Linux it is /proc/net/unix's Num
	// column with an "0x" in front, which identifies one only where the kernel is
	// willing to print its pointers, so the Inode is preferred there; see
	// PeerInode and unixSocketIdentity().
	//
	// Comparing the text is enough, lsof padding neither this nor PeerDevice:
	// 94.7% of distinct devices and 94.8% of distinct peers are 16 hex digits,
	// the same distribution, and of 489 peers measured on a quiet macOS laptop
	// none resolved numerically but not textually. Two hex numbers of different
	// lengths are just two numbers of different magnitudes.
	Device string

	// lsof's "i" column, this socket's inode, "14602". Linux only — macOS lsof
	// reports no inode for a unix socket, and asking for the field there costs
	// nothing.
	//
	// What joins lsof's records to the netlink dump, which is keyed on the inode
	// and knows nothing of kernel addresses, and what identifies a socket on
	// Linux once they are joined; see unixSocketIdentity().
	Inode string

	// The Device of the socket at the other end, from lsof's "n->0x..." name.
	// macOS only — a Linux peer arrives from the netlink dump as an inode and
	// stays one, see PeerInode.
	//
	// Empty for a socket with no peer, which is a listener, a socket nobody
	// dialed, or one whose peer is gone — and also every socket carrying a Path,
	// lsof naming those by the path instead. A peer with no Device gets no line.
	PeerDevice string

	// The inode of the socket at the other end, "14602". Linux only, straight
	// from the netlink dump, which names a peer by its inode and knows nothing of
	// kernel addresses.
	//
	// This rather than PeerDevice is what identifies a peer on Linux, lsof's
	// device being no identity there whenever the kernel withholds its pointers.
	// /proc/net/unix prints the address that column comes from with "%pK", which
	// the kernel fills with zeroes for a reader without CAP_SYSLOG under
	// kernel.kptr_restrict=1 — what Ubuntu ships in
	// /etc/sysctl.d/10-kernel-hardening.conf — and for every reader at all under
	// kptr_restrict=2. Measured non-root in a container: nine sockets, nine
	// inodes, one device between them.
	//
	// Empty for a socket with no peer, which is a listener, one nobody dialed, or
	// one whose peer is gone. A peer we aren't allowed to see is named here all
	// the same, and left out by the matching finding no socket of that inode.
	PeerInode string

	// The path this socket is bound to, "/tmp/probe.sock", or "@name" for one in
	// the abstract namespace, which is Linux only. Carried by a listener and by
	// every socket accepted on it, empty for both ends of a socketpair(2), and
	// empty for a socket that dialed one of those unless it bound an address of
	// its own first.
	Path string

	// Whether listen(2) was called on this socket, which is what makes its Path a
	// service address rather than an address somebody bound to be replied to.
	//
	// Linux only, from the netlink dump's udiag_state. Always false on macOS,
	// where lsof reports no such thing — and where nothing needs it, a socket
	// there never being reported with a peer and a Path both; see
	// UnixSocketConnections().
	//
	// True on the listening socket alone, never on the sockets accepted from it,
	// so this is no use read off the two ends of a connection: neither of those
	// ends is the listener. What it identifies is the Path, and a service Path is
	// the one some socket in the listing listens on.
	//
	// False for a datagram socket however much of a service it is: listen(2) is a
	// stream and seqpacket call, so /dev/log has nothing to set this by. Measured
	// in a container: a datagram server carries no SO_ACCEPTCON.
	Listening bool
}

// Maps PIDs to the unix domain sockets held open by the corresponding
// processes.
//
// The listing is partial: processes we aren't allowed to inspect are missing
// from the map, so expect only a fraction of the running processes when not
// running as root. Processes without any unix sockets are missing as well.
//
// This forks lsof, which takes a fraction of a second, and on Linux asks the
// kernel for a netlink dump on top of that. Too slow for calling once per frame,
// fine for on-demand lookups.
func GetUnixSocketsByPid() (map[int][]UnixSocket, error) {
	parser := newLsofUnixSocketParser()

	// -n: Don't resolve host names. A unix socket has none to resolve, so this
	//   only makes sure lsof never goes looking.
	// -w: Don't warn about processes we aren't allowed to inspect
	// -U: List unix domain sockets only, which is the filter pipes have no
	//   equivalent of. So this listing costs what the "-i" one in sockets.go
	//   does rather than what the unfiltered one in GetPipeEndsByPid() does.
	// -F pfndi0: Machine readable output with NUL terminated PID, file
	//   descriptor, name, device and inode fields. The device is this socket's
	//   own kernel address, and on macOS the name is the peer's — that pair being
	//   what the peer matching in UnixSocketConnections() is built on. Linux
	//   reports no peer and no path, so there the inode is what
	//   fillInPeersAndPaths() joins the netlink dump on; macOS reports no inode
	//   and asking costs nothing.
	commandline := []string{"lsof", "-n", "-w", "-U", "-F", "pfndi0"}

	// Locale intentionally left alone, matching GetCwdsByPid()
	err := util.ExecInUsersLocale(commandline, parser.parseLine)
	if err != nil {
		// A machine holding no unix socket at all is an empty listing rather than
		// a failure, the way an idle machine is for GetSocketsByPid(), so the exit
		// status alone doesn't fail this the way it does the pipe listing. The cost
		// is that an lsof failing in no other way passes for that too.
		//
		// Defensive rather than known to be needed: "-i" makes lsof exit 1 whenever
		// it locates no internet socket, but "-U" over a container holding no unix
		// socket at all printed nothing and exited 0 on lsof 4.99.4. Whether some
		// other lsof counts "-U" as a search item the way it counts "-i" is
		// untested.
		if !util.IsExitStatus(err) && len(parser.unixSocketsByPid) == 0 {
			// Something other than a non-zero exit code from lsof, this is a real
			// problem.
			return nil, err
		}

		log.Infof("Kept %d processes' worth of unix sockets despite: %v",
			len(parser.unixSocketsByPid), err)
	}

	err = fillInPeersAndPaths(parser.unixSocketsByPid)
	if err != nil {
		return nil, err
	}

	return parser.unixSocketsByPid, nil
}

// Parses the output of "lsof -n -w -U -F pfndi0", which comes in NUL terminated
// fields, one line per process and then one line per socket. On macOS:
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
// Those are the only three shapes a macOS name comes in, and a record has a peer
// or a path and never both: measured on a quiet macOS laptop, 551 records, 462
// with a peer, 82 with a path, none with the two of them.
//
// Linux reports the same connected pair like this, no peer anywhere and an inode
// on every record:
//
//	p250\0
//	f4\0d0x00000000d53f7360\0i14598\0n/tmp/probe.sock type=STREAM\0
//	f5\0d0x000000005b7e5c5a\0i14602\0ntype=STREAM\0
//	f13\0d0x0000000060561eb9\0i14609\0n/tmp/probe.sock type=STREAM\0
//
// The listener, the client's end and the accepted end, in that order. What that
// leaves out is filled in from a netlink dump; see fillInPeersAndPaths().
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

// Note that lsof escapes non-printable characters, newlines included, so one
// line of output is always one complete record.
func (parser *lsofUnixSocketParser) parseLine(line string) error {
	// Filled in by the fields of this line, whether or not it turns out to
	// describe a socket
	var record lsofUnixSocketRecord

	for field := range strings.SplitSeq(strings.TrimSuffix(line, "\x00"), "\x00") {
		err := parser.parseField(field, &record)
		if err != nil {
			return err
		}
	}

	if !record.isASocket {
		// A line naming the process the sockets below it belong to. With "-U"
		// there is nothing else it can be.
		return nil
	}

	if parser.pid == -1 {
		return fmt.Errorf("lsof reported unix socket on fd <%s> before any PID", record.socket.Fd)
	}

	parser.unixSocketsByPid[parser.pid] = append(parser.unixSocketsByPid[parser.pid], record.socket)

	return nil
}

// One lsof record being decoded field by field, whether it describes a socket or
// the process holding one.
type lsofUnixSocketRecord struct {
	socket UnixSocket

	// True once a file descriptor field has arrived, which is what tells a socket
	// apart from the process line above it
	isASocket bool
}

// Applies one field to record, or to the parser itself for the PID field.
func (parser *lsofUnixSocketParser) parseField(field string, record *lsofUnixSocketRecord) error {
	if field == "" {
		return nil
	}

	identifier := field[0]
	value := field[1:]

	switch identifier {
	case 'p':
		pid, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("unparseable lsof PID <%s>: %w", value, err)
		}

		parser.pid = pid

	case 'f':
		record.socket.Fd = value
		record.isASocket = true

	case 'd':
		record.socket.Device = value

	case 'i':
		record.socket.Inode = value

	case 'n':
		applyUnixSocketName(value, &record.socket)
	}

	// lsof can emit fields we didn't ask for, just ignore those
	return nil
}

// Applies lsof's name column to socket, which on macOS is one of three things:
// the kernel address of the socket at the other end, spelled "->0x...", the path
// this socket is bound to, or "->(none)" for a socket whose peer is gone.
//
// A name is a peer when it starts with "->" and a path in every other case. Not
// every path is absolute — a socket bound to a relative one is named by just
// that, with no leading slash to recognize it by — so testing for a slash
// instead would drop those.
//
// Linux spells the name a fourth way this makes no sense of, and knowingly:
// "/tmp/probe.sock type=STREAM" for a bound socket and a bare "type=STREAM" for
// one that isn't, verified against a real connected pair on lsof 4.99.4. So a
// Linux Path comes out with that suffix on it, or holding nothing but the suffix,
// and fillInPeersAndPaths() overwrites it with the netlink dump's name rather
// than anybody cutting the suffix off. One source for the path and the peer both,
// and no per-type suffix to keep up with.
func applyUnixSocketName(name string, socket *UnixSocket) {
	peerDevice, namesAPeer := strings.CutPrefix(name, "->")
	if !namesAPeer {
		socket.Path = name
		return
	}

	if peerDevice == "(none)" {
		// Whoever was at the other end is gone, so there is nobody to name
		return
	}

	socket.PeerDevice = peerDevice
}
