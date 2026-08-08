package processes

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// From linux/unix_diag.h, which golang.org/x/sys/unix stops short of: it has
// NETLINK_SOCK_DIAG and SOCK_DIAG_BY_FAMILY and nothing about the unix socket
// dump those two are for.
const (
	// udiag_show bits, saying which attributes a dump should come back with
	udiagShowName = 0x1
	udiagShowPeer = 0x4

	// The attribute types those two bits ask for
	unixDiagName = 0
	unixDiagPeer = 2
)

// linux/tcp_states.h's TCP_LISTEN, which is the udiag_state of a unix socket
// listen(2) was called on. The states are TCP's whatever the family, and a unix
// socket only ever reports this one or TCP_ESTABLISHED or TCP_CLOSE.
//
// Measured in a container: dbus-daemon's /run/dbus/system_bus_socket came back
// with 10 while the two sockets accepted on it came back with 1, and so did the
// abstract addresses two sd-bus clients had bound for themselves.
const tcpListen = 10

// linux/unix_diag.h's struct unix_diag_req, the body of a dump request.
//
// Laid out to match that struct byte for byte, which it does on every
// architecture we build for: the fields are naturally aligned in the order they
// are declared, so no padding creeps in between them.
type unixDiagReq struct {
	Family   uint8
	Protocol uint8
	Pad      uint16
	States   uint32
	Ino      uint32
	Show     uint32
	Cookie   [2]uint32
}

// linux/unix_diag.h's struct unix_diag_msg, the fixed size head of every record
// a dump answers with. The attributes we asked for follow it.
type unixDiagMsg struct {
	Family uint8
	Type   uint8
	State  uint8
	Pad    uint8
	Ino    uint32
	Cookie [2]uint32
}

// Fills in the peer and the path of every socket in unixSocketsByPid, neither of
// which Linux lsof can report: its name field holds "type=STREAM" rather than a
// path, and it knows no peer at all. Both come from a sock_diag netlink dump
// instead, and both get overwritten with what that dump says — so a socket the
// dump has no record of comes out with no peer and no path, which is what we
// know about it.
//
// The peer comes out named by its inode, which is how the dump names one and how
// UnixSocketConnections() identifies a socket wherever there are inodes to be had.
// Translating it into the Device that identifies a socket on macOS would be the
// wrong move: that column is all zeroes on a machine whose kernel withholds its
// pointers, and the inodes are what stays distinct there; see
// UnixSocket.PeerInode.
//
// Inodes are unique per socket rather than per process, so one holder naming a
// peer says the same as any other. They are unique across network namespaces as
// well, which matters because the dump covers ours while lsof lists every process
// on the machine: sockfs numbers its inodes off one global counter, so a peer
// inode of ours cannot name a container's socket by accident. That much is
// reasoning about the kernel rather than something measured here.
//
// A peer we aren't allowed to see gets named all the same, there being nothing to
// gain by dropping it here. The matching leaves it out instead, finding no socket
// of that inode in a listing that skips the processes we cannot inspect — the way
// an invisible peer already degrades on macOS.
//
// Fails if the dump does, there being nothing worth showing without it: sockets
// with no peers make no connections, so what would render is an empty IPC
// section rather than the error.
func fillInPeersAndPaths(unixSocketsByPid map[int][]UnixSocket) error {
	peersByInode, err := unixSocketPeersByInode()
	if err != nil {
		return err
	}

	for _, sockets := range unixSocketsByPid {
		for i := range sockets {
			socket := &sockets[i]

			// The zero value for a socket the dump doesn't mention, which clears
			// both fields rather than leaving lsof's "type=STREAM" in the path
			peer := peersByInode[socket.Inode]

			socket.Path = peer.path
			socket.PeerInode = peer.peerInode
			socket.Listening = peer.listening
		}
	}

	return nil
}

// What a sock_diag dump knows about one unix domain socket that lsof does not.
type unixSocketPeer struct {
	// The inode of the socket at the other end, "379". Empty for a socket that
	// has no peer: a listener, or one that was never connected.
	//
	// The pairing is symmetric for stream and seqpacket sockets, unlike the naming
	// on macOS: both ends of such a connected pair name each other, listeners
	// excepted. A connected datagram socket, /dev/log's clients being the everyday
	// ones, names its server and is not named back — sk_diag_dump_peer() reporting
	// what unix_peer() holds, which a server never sets.
	peerInode string

	// The path this socket is bound to, "/tmp/probe.sock", or "@name" for one in
	// the abstract namespace. Empty for the socket that dialed such a path and
	// empty for both ends of a socketpair(2).
	path string

	// Whether listen(2) was called on this socket, from udiag_state; see
	// UnixSocket.Listening for what it is good for.
	listening bool
}

// Every unix domain socket in our network namespace, keyed by inode, as a
// sock_diag netlink dump reports them.
//
// This is where the peer edge comes from, there being no other source for it:
// lsof and /proc/net/unix both report a socket's own kernel address and stop
// there, and "ss -x" gets its peer column from this same dump. Asking for the
// dump ourselves is a socket and a parse rather than another fork, and it works
// unprivileged where lsof degrades — re-running it as "nobody" in a container
// gave byte identical output, peers included.
//
// Netlink carries no PID, which is why this supplies the peer edge only and lsof
// stays the process/descriptor/inode source. "ss -p" gets its PIDs by walking
// /proc/*/fd itself, and we could too — the fd symlinks spell out "socket:[378]",
// which is the same join with no fork at all. lsof keeps that job because the
// record shape and the parser are then shared with macOS, which has no /proc to
// walk, where a Linux-only collector would be a second code path for the half of
// the data both platforms already agree on. Revisit if lsof turns out to be the
// slow part, not to save the fork.
//
// /proc/net/unix cannot stand in for the dump either, and neither can px's
// approach on top of it. That file has no peer column at all, and its Num field is
// the socket's own kernel address — the same number lsof prints as the device, so
// lsof's device field is this file reformatted:
//
//	Num       RefCount Protocol Flags    Type St Inode Path
//	00000000609cd1bd: 00000003 00000000 00000000 0001 03   378                   <- client end
//	000000001942c573: 00000003 00000000 00000000 0001 03   379 /tmp/probe.sock   <- accepted end
//	00000000ee2a9915: 00000002 00000000 00010000 0001 01   369 /tmp/probe.sock   <- listener
//
// The client's connected socket carries no path, which is what sinks px's Linux
// fallback: its device_number to files-with-the-same-name matching
// (px_ipc_map.py:260-267) can only relate processes that share a path, so these
// land in its "UNKNOWN destinations: Running with sudo might help" bucket
// (px_ipc_map.py:152-153) however privileged it runs.
func unixSocketPeersByInode() (map[string]unixSocketPeer, error) {
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.NETLINK_SOCK_DIAG)
	if err != nil {
		return nil, fmt.Errorf("opening a sock_diag netlink socket: %w", err)
	}

	defer func() { _ = unix.Close(fd) }()

	err = requestUnixSocketDump(fd)
	if err != nil {
		return nil, err
	}

	peersByInode := map[string]unixSocketPeer{}

	// A dump arrives in chunks of several records each, and this is sized so that
	// a chunk always fits whole: netlink_dump() grows its allocation to the
	// largest recvmsg the socket has seen, and netlink_recvmsg() caps what it
	// remembers at SKB_WITH_OVERHEAD(32768) — just under 32 KiB, whatever a reader
	// asks for. That cap is what makes this number enough rather than the number
	// itself.
	//
	// Nothing much to gain by shrinking it, and a floor to respect if anybody
	// tries: the first chunk is sized before the kernel has seen a recvmsg at all,
	// to NLMSG_GOODSIZE, which is a page or 8 KiB wherever pages are bigger. A
	// buffer under that truncates, and truncation is silent from here — unix.Read
	// reports no MSG_TRUNC, and netlinkAttributes() makes what it can of a
	// truncated tail by design, so the lost records would surface as missing
	// connections rather than as an error.
	buffer := make([]byte, 32*1024)

	for {
		count, err := unix.Read(fd, buffer)
		if errors.Is(err, unix.EINTR) {
			// A signal arrived, and the Go runtime sends itself plenty of those to
			// preempt goroutines with. Nothing was read, so just ask again.
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("reading the sock_diag dump: %w", err)
		}

		messages, err := syscall.ParseNetlinkMessage(buffer[:count])
		if err != nil {
			return nil, fmt.Errorf("parsing the sock_diag dump: %w", err)
		}

		for _, message := range messages {
			if message.Header.Type == unix.NLMSG_DONE {
				return peersByInode, nil
			}

			if message.Header.Type == unix.NLMSG_ERROR {
				return nil, netlinkError(message.Data)
			}

			inode, peer, err := parseUnixDiagRecord(message.Data)
			if err != nil {
				return nil, err
			}

			peersByInode[inode] = peer
		}
	}
}

// Asks fd for every unix domain socket there is, with its name and its peer.
func requestUnixSocketDump(fd int) error {
	request := unixDiagReq{
		Family: unix.AF_UNIX,
		// Every state there is, so that listeners come along too. They have no
		// peer, but their inode is what makes a client naming one nameable.
		States: ^uint32(0),
		Show:   udiagShowName | udiagShowPeer,
	}

	header := unix.NlMsghdr{
		Len:   uint32(unix.SizeofNlMsghdr) + uint32(unsafe.Sizeof(request)),
		Type:  unix.SOCK_DIAG_BY_FAMILY,
		Flags: unix.NLM_F_REQUEST | unix.NLM_F_DUMP,
	}

	message := make([]byte, 0, header.Len)
	message = append(message, unsafe.Slice((*byte)(unsafe.Pointer(&header)), unix.SizeofNlMsghdr)...)
	message = append(message, unsafe.Slice((*byte)(unsafe.Pointer(&request)), unsafe.Sizeof(request))...)

	err := unix.Sendto(fd, message, 0, &unix.SockaddrNetlink{Family: unix.AF_NETLINK})
	if err != nil {
		return fmt.Errorf("asking sock_diag for a unix socket dump: %w", err)
	}

	return nil
}

// Decodes one record of a unix socket dump into the inode it is about and what
// the dump says about that socket.
func parseUnixDiagRecord(data []byte) (string, unixSocketPeer, error) {
	headSize := int(unsafe.Sizeof(unixDiagMsg{}))
	if len(data) < headSize {
		return "", unixSocketPeer{}, fmt.Errorf("sock_diag record of %d bytes is too short", len(data))
	}

	head := (*unixDiagMsg)(unsafe.Pointer(&data[0]))
	inode := strconv.FormatUint(uint64(head.Ino), 10)

	peer := unixSocketPeer{listening: head.State == tcpListen}
	for _, attribute := range netlinkAttributes(data[headSize:]) {
		switch attribute.kind {
		case unixDiagPeer:
			if len(attribute.value) < 4 {
				return "", unixSocketPeer{}, fmt.Errorf(
					"sock_diag peer of %d bytes is too short", len(attribute.value))
			}

			peer.peerInode = strconv.FormatUint(uint64(binary.NativeEndian.Uint32(attribute.value)), 10)

		case unixDiagName:
			peer.path = unixSocketPath(string(attribute.value))
		}
	}

	return inode, peer, nil
}

// The path a UNIX_DIAG_NAME attribute names, in the spelling lsof and
// /proc/net/unix use for the same socket.
//
// The kernel hands out the sockaddr as it was bound, so a path comes with the
// terminating NUL along and a socket in the abstract namespace comes as a
// leading NUL followed by its name — "/tmp/probe.sock\x00" and
// "\x00ftop-probe-abstract", both verified in a container. Everything else
// spells the abstract one "@ftop-probe-abstract".
func unixSocketPath(name string) string {
	abstract, isAbstract := strings.CutPrefix(name, "\x00")
	if isAbstract {
		return "@" + abstract
	}

	return strings.TrimSuffix(name, "\x00")
}

// The error an NLMSG_ERROR message carries, which is a negative errno at the
// head of linux/netlink.h's struct nlmsgerr.
func netlinkError(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("sock_diag failed, and its error of %d bytes is too short to read", len(data))
	}

	errno := syscall.Errno(-int32(binary.NativeEndian.Uint32(data)))

	return fmt.Errorf("sock_diag refused a unix socket dump: %w", errno)
}

// One netlink attribute, the TLV that linux/netlink.h's struct rtattr heads.
type netlinkAttribute struct {
	kind  uint16
	value []byte
}

// The attributes in data, which is whatever follows a netlink record's fixed
// size head.
//
// Anything that doesn't parse ends the list rather than failing: a truncated
// attribute is one we can make nothing of, and the ones before it are still
// good.
func netlinkAttributes(data []byte) []netlinkAttribute {
	var attributes []netlinkAttribute

	for len(data) >= unix.SizeofRtAttr {
		head := (*unix.RtAttr)(unsafe.Pointer(&data[0]))
		if int(head.Len) < unix.SizeofRtAttr || int(head.Len) > len(data) {
			break
		}

		attributes = append(attributes, netlinkAttribute{
			kind:  head.Type,
			value: data[unix.SizeofRtAttr:head.Len],
		})

		// Every attribute starts on a four byte boundary, so the padding after
		// one that didn't end on one belongs to nobody
		aligned := (int(head.Len) + 3) & ^3
		if aligned > len(data) {
			break
		}

		data = data[aligned:]
	}

	return attributes
}
