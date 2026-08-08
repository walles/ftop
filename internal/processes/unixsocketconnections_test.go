package processes

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/walles/ftop/internal/assert"
)

// A client's socket names the socket the server accepted it on, and that one
// carries the path the connection was made over. The client is the end doing the
// naming, so it is the one that dialed.
func TestUnixSocketConnections_client(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "curl"}
	server := &Process{Pid: 1, Cmdline: "dockerd"}

	unixSockets := map[int][]UnixSocket{
		42: {{Fd: "3", Device: "0x3333", PeerDevice: "0x2222"}},
		1: {
			{Fd: "3", Device: "0x1111", Path: "/var/run/docker.sock"},
			{Fd: "4", Device: "0x2222", Path: "/var/run/docker.sock"},
		},
	}

	connections := UnixSocketConnections(me, []*Process{me, server}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "dockerd", Pid: 1},
			Protocol:  ProtocolUnix,
			Direction: DirectionOutgoing,
			Path:      "/var/run/docker.sock",
			Count:     1,
		},
	})
}

// The same connection from the server's side: our accepted socket knows the path
// but not who dialed it, so a client naming it is what finds us a peer at all.
//
// Our listening socket, which nobody named, gets no line. It is not a connection.
func TestUnixSocketConnections_server(t *testing.T) {
	me := &Process{Pid: 1, Cmdline: "dockerd"}
	client := &Process{Pid: 42, Cmdline: "curl"}

	unixSockets := map[int][]UnixSocket{
		42: {{Fd: "3", Device: "0x3333", PeerDevice: "0x2222"}},
		1: {
			{Fd: "3", Device: "0x1111", Path: "/var/run/docker.sock"},
			{Fd: "4", Device: "0x2222", Path: "/var/run/docker.sock"},
		},
	}

	connections := UnixSocketConnections(me, []*Process{me, client}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "curl", Pid: 42},
			Protocol:  ProtocolUnix,
			Direction: DirectionIncoming,
			Path:      "/var/run/docker.sock",
			Count:     1,
		},
	})
}

// The same client and server as reported on Linux, where a netlink peer edge is
// symmetric: both ends name each other, so the naming says nothing about who
// dialed and the arrow comes off the paths instead.
func TestUnixSocketConnections_linuxClient(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "curl"}
	server := &Process{Pid: 1, Cmdline: "dockerd"}

	unixSockets := map[int][]UnixSocket{
		42: {{Fd: "3", Device: "0x3333", PeerDevice: "0x2222"}},
		1: {
			{Fd: "3", Device: "0x1111", Path: "/var/run/docker.sock"},
			{Fd: "4", Device: "0x2222", PeerDevice: "0x3333", Path: "/var/run/docker.sock"},
		},
	}

	connections := UnixSocketConnections(me, []*Process{me, server}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "dockerd", Pid: 1},
			Protocol:  ProtocolUnix,
			Direction: DirectionOutgoing,
			Path:      "/var/run/docker.sock",
			Count:     1,
		},
	})
}

// That same Linux listing from the server's side, which is one connection and not
// two however many ends of it name each other.
func TestUnixSocketConnections_linuxServer(t *testing.T) {
	me := &Process{Pid: 1, Cmdline: "dockerd"}
	client := &Process{Pid: 42, Cmdline: "curl"}

	unixSockets := map[int][]UnixSocket{
		42: {{Fd: "3", Device: "0x3333", PeerDevice: "0x2222"}},
		1: {
			{Fd: "3", Device: "0x1111", Path: "/var/run/docker.sock"},
			{Fd: "4", Device: "0x2222", PeerDevice: "0x3333", Path: "/var/run/docker.sock"},
		},
	}

	connections := UnixSocketConnections(me, []*Process{me, client}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "curl", Pid: 42},
			Protocol:  ProtocolUnix,
			Direction: DirectionIncoming,
			Path:      "/var/run/docker.sock",
			Count:     1,
		},
	})
}

// lsof's device column for every unix socket on a machine whose kernel withholds
// its pointers, read off "lsof -n -w -U -F pfndi0" under kernel.kptr_restrict=1
// in a container. /proc/net/unix prints the kernel address that column comes from
// with "%pK", which is this for any reader without CAP_SYSLOG, so every socket on
// such a machine reports the same device and none of them an identity.
//
// Sixteen digits because the kernel zero pads "%pK" to twice a pointer's width,
// and lsof passes that text through with an "0x" in front of it rather than
// reformatting it. A 32 bit kernel spells the same thing "0x00000000", which
// nothing here needs to know: these tests need every socket to report the same
// device, not a particular one.
const restrictedDevice = "0x0000000000000000"

// A client and its server on a machine that withholds kernel pointers, which is
// what Ubuntu ships. The devices are all the same and identify nothing, so the
// inodes are what the two ends of the connection are found by.
//
// Every Linux record carries an inode, lsof's "i" column, and the netlink dump
// names a peer by inode as well — so the whole matching can be done without ever
// consulting a device, which is what makes this listing come out the same as an
// unrestricted one.
func TestUnixSocketConnections_linuxRestrictedKernelPointers(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "curl"}
	server := &Process{Pid: 1, Cmdline: "dockerd"}

	unixSockets := map[int][]UnixSocket{
		42: {{Fd: "3", Device: restrictedDevice, Inode: "7423", PeerInode: "2201"}},
		1: {
			{Fd: "3", Device: restrictedDevice, Inode: "2196", Path: "/var/run/docker.sock"},
			{Fd: "4", Device: restrictedDevice, Inode: "2201", PeerInode: "7423", Path: "/var/run/docker.sock"},
		},
	}

	connections := UnixSocketConnections(me, []*Process{me, server}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "dockerd", Pid: 1},
			Protocol:  ProtocolUnix,
			Direction: DirectionOutgoing,
			Path:      "/var/run/docker.sock",
			Count:     1,
		},
	})
}

// A connection only the other end names, on that same machine: our socket names
// nobody, so the one line it deserves is found by the scan for sockets naming
// ours rather than by looking ours up.
//
// Which is a second thing to get right rather than the first one again. A
// connection our socket names is an index lookup, and this is the other loop —
// the one that has to recognize a socket of ours as the peer another socket
// names, and so has to agree with that lookup about what identifies a socket.
//
// This is a datagram server, /dev/log being the everyday one: the netlink peer
// edge is symmetric for stream and seqpacket sockets, but a connected datagram
// socket names its server while the server names nobody back. So it is the shape
// that reaches this loop and no other, on Linux, however unrestricted the kernel
// pointers are.
func TestUnixSocketConnections_linuxRestrictedKernelPointersNamedByPeer(t *testing.T) {
	me := &Process{Pid: 1, Cmdline: "syslogd"}
	client := &Process{Pid: 42, Cmdline: "cron"}

	unixSockets := map[int][]UnixSocket{
		42: {{Fd: "3", Device: restrictedDevice, Inode: "7423", PeerInode: "2196"}},
		1:  {{Fd: "3", Device: restrictedDevice, Inode: "2196", Path: "/dev/log"}},
	}

	connections := UnixSocketConnections(me, []*Process{me, client}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "cron", Pid: 42},
			Protocol:  ProtocolUnix,
			Direction: DirectionIncoming,
			Path:      "/dev/log",
			Count:     1,
		},
	})
}

// A process holding nothing but a socketpair of its own, on that same machine,
// alongside two processes connected to each other. It gets the one line its
// socketpair is worth and no line about them.
//
// The listing is what makes this worth pinning: the devices say all five sockets
// are the same socket, so a peer looked up by device lands on whichever of them
// the listing happened to keep — a stranger, over a path this process neither
// serves nor dialed. Its own socketpair is the answer, and both ends of it are
// held right here.
func TestUnixSocketConnections_linuxRestrictedKernelPointersUnrelatedProcess(t *testing.T) {
	me := &Process{Pid: 99, Cmdline: "socat"}
	client := &Process{Pid: 42, Cmdline: "curl"}
	server := &Process{Pid: 1, Cmdline: "dockerd"}

	unixSockets := map[int][]UnixSocket{
		99: {
			{Fd: "5", Device: restrictedDevice, Inode: "16660", PeerInode: "16661"},
			{Fd: "6", Device: restrictedDevice, Inode: "16661", PeerInode: "16660"},
		},
		42: {{Fd: "3", Device: restrictedDevice, Inode: "7423", PeerInode: "2201"}},
		1: {
			{Fd: "3", Device: restrictedDevice, Inode: "2196", Path: "/var/run/docker.sock"},
			{Fd: "4", Device: restrictedDevice, Inode: "2201", PeerInode: "7423", Path: "/var/run/docker.sock"},
		},
	}

	connections := UnixSocketConnections(me, []*Process{me, client, server}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "socat", Pid: 99},
			Protocol:  ProtocolUnix,
			Direction: DirectionUnknown,
			Count:     1,
		},
	})
}

// A server offering two services holds a socket on each path, and the path a
// connection gets is the one carried by the socket the client actually named —
// not whichever of ours happens to have one.
//
// The socket on /run/a.sock is a listener nobody has dialed, so it contributes
// no line and no path.
func TestUnixSocketConnections_serverWithTwoPaths(t *testing.T) {
	me := &Process{Pid: 1, Cmdline: "dockerd"}
	client := &Process{Pid: 42, Cmdline: "curl"}

	unixSockets := map[int][]UnixSocket{
		42: {{Fd: "3", Device: "0x3333", PeerDevice: "0x2222"}},
		1: {
			{Fd: "3", Device: "0x1111", Path: "/run/a.sock"},
			{Fd: "4", Device: "0x2222", Path: "/run/b.sock"},
		},
	}

	connections := UnixSocketConnections(me, []*Process{me, client}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "curl", Pid: 42},
			Protocol:  ProtocolUnix,
			Direction: DirectionIncoming,
			Path:      "/run/b.sock",
			Count:     1,
		},
	})
}

// A client can end up naming the listening socket rather than the socket it was
// accepted on, and that one carries the same path. Either way the peer is the
// server, so this looks no different from the outside.
func TestUnixSocketConnections_clientNamingTheListener(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "curl"}
	server := &Process{Pid: 1, Cmdline: "dockerd"}

	unixSockets := map[int][]UnixSocket{
		42: {{Fd: "3", Device: "0x3333", PeerDevice: "0x1111"}},
		1:  {{Fd: "3", Device: "0x1111", Path: "/var/run/docker.sock"}},
	}

	connections := UnixSocketConnections(me, []*Process{me, server}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "dockerd", Pid: 1},
			Protocol:  ProtocolUnix,
			Direction: DirectionOutgoing,
			Path:      "/var/run/docker.sock",
			Count:     1,
		},
	})
}

// The two ends of a socketpair(2) name one another and neither carries a path.
// Both of them dialed, which is to say neither did, so there is no telling which
// way to point an arrow.
func TestUnixSocketConnections_socketPair(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "parent"}
	peer := &Process{Pid: 5678, Cmdline: "child"}

	unixSockets := map[int][]UnixSocket{
		1234: {{Fd: "5", Device: "0xaaaa", PeerDevice: "0xbbbb"}},
		5678: {{Fd: "0", Device: "0xbbbb", PeerDevice: "0xaaaa"}},
	}

	connections := UnixSocketConnections(me, []*Process{me, peer}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "child", Pid: 5678},
			Protocol:  ProtocolUnix,
			Direction: DirectionUnknown,
			Count:     1,
		},
	})
}

// Most peers on a non-root listing are held by processes we aren't allowed to
// inspect, and a peer we cannot find is one we cannot name. Those get no line,
// which is how a pipe whose peer is gone already degrades.
func TestUnixSocketConnections_peerNotFound(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "curl"}

	unixSockets := map[int][]UnixSocket{
		42: {{Fd: "3", Device: "0x3333", PeerDevice: "0xdead"}},
	}

	connections := UnixSocketConnections(me, []*Process{me}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection(nil))
}

// A socket whose peer is gone names nobody, and there is nothing to say about
// it.
func TestUnixSocketConnections_noPeerAtAll(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "curl"}

	unixSockets := map[int][]UnixSocket{
		42: {{Fd: "0", Device: "0x3333"}},
	}

	connections := UnixSocketConnections(me, []*Process{me}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection(nil))
}

// One socket open on several file descriptors, by dup(2) or by being inherited
// as more than one of the standard three, is reported once per descriptor. That
// is one connection however many times it arrives.
func TestUnixSocketConnections_duplicatedDescriptors(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "curl"}
	server := &Process{Pid: 1, Cmdline: "dockerd"}

	unixSockets := map[int][]UnixSocket{
		42: {
			{Fd: "1", Device: "0x3333", PeerDevice: "0x2222"},
			{Fd: "4", Device: "0x3333", PeerDevice: "0x2222"},
		},
		1: {
			{Fd: "4", Device: "0x2222", Path: "/var/run/docker.sock"},
			{Fd: "7", Device: "0x2222", Path: "/var/run/docker.sock"},
		},
	}

	connections := UnixSocketConnections(me, []*Process{me, server}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "dockerd", Pid: 1},
			Protocol:  ProtocolUnix,
			Direction: DirectionOutgoing,
			Path:      "/var/run/docker.sock",
			Count:     1,
		},
	})
}

// Several connections to one peer over one path are one line, counting them.
func TestUnixSocketConnections_severalConnectionsToOnePeer(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "curl"}
	server := &Process{Pid: 1, Cmdline: "dockerd"}

	unixSockets := map[int][]UnixSocket{
		42: {
			{Fd: "3", Device: "0x3333", PeerDevice: "0x2222"},
			{Fd: "4", Device: "0x4444", PeerDevice: "0x5555"},
		},
		1: {
			{Fd: "4", Device: "0x2222", Path: "/var/run/docker.sock"},
			{Fd: "5", Device: "0x5555", Path: "/var/run/docker.sock"},
		},
	}

	connections := UnixSocketConnections(me, []*Process{me, server}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "dockerd", Pid: 1},
			Protocol:  ProtocolUnix,
			Direction: DirectionOutgoing,
			Path:      "/var/run/docker.sock",
			Count:     2,
		},
	})
}

// Two paths to one peer are two different services of that peer, so they get a
// line each rather than a count of two. Sorted by path, so that the same peer's
// lines come out in the same order every time.
func TestUnixSocketConnections_twoPathsToOnePeer(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "curl"}
	server := &Process{Pid: 1, Cmdline: "dockerd"}

	unixSockets := map[int][]UnixSocket{
		42: {
			{Fd: "3", Device: "0x3333", PeerDevice: "0x2222"},
			{Fd: "4", Device: "0x4444", PeerDevice: "0x5555"},
		},
		1: {
			{Fd: "4", Device: "0x2222", Path: "/var/run/docker.sock"},
			{Fd: "5", Device: "0x5555", Path: "/var/run/backend.sock"},
		},
	}

	connections := UnixSocketConnections(me, []*Process{me, server}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "dockerd", Pid: 1},
			Protocol:  ProtocolUnix,
			Direction: DirectionOutgoing,
			Path:      "/var/run/backend.sock",
			Count:     1,
		},
		{
			Peer:      Peer{Name: "dockerd", Pid: 1},
			Protocol:  ProtocolUnix,
			Direction: DirectionOutgoing,
			Path:      "/var/run/docker.sock",
			Count:     1,
		},
	})
}

// One listener serving many clients is one line per client, since each of them
// is a process of its own to talk to.
func TestUnixSocketConnections_severalClients(t *testing.T) {
	me := &Process{Pid: 1, Cmdline: "dockerd"}
	first := &Process{Pid: 42, Cmdline: "curl"}
	second := &Process{Pid: 43, Cmdline: "compose"}

	unixSockets := map[int][]UnixSocket{
		42: {{Fd: "3", Device: "0x3333", PeerDevice: "0x2222"}},
		43: {{Fd: "3", Device: "0x4444", PeerDevice: "0x5555"}},
		1: {
			{Fd: "4", Device: "0x2222", Path: "/var/run/docker.sock"},
			{Fd: "5", Device: "0x5555", Path: "/var/run/docker.sock"},
		},
	}

	connections := UnixSocketConnections(me, []*Process{me, first, second}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "compose", Pid: 43},
			Protocol:  ProtocolUnix,
			Direction: DirectionIncoming,
			Path:      "/var/run/docker.sock",
			Count:     1,
		},
		{
			Peer:      Peer{Name: "curl", Pid: 42},
			Protocol:  ProtocolUnix,
			Direction: DirectionIncoming,
			Path:      "/var/run/docker.sock",
			Count:     1,
		},
	})
}

// A listening socket inherited across a fork is held by parent and child alike,
// and a client naming it has no way of saying which of them it is talking to.
// The lowest PID wins, the way it does for an inherited TCP socket: arbitrary,
// but the same answer every time the page is opened.
func TestUnixSocketConnections_inheritedListener(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "curl"}
	parent := &Process{Pid: 1, Cmdline: "dockerd"}
	child := &Process{Pid: 2, Cmdline: "dockerd"}

	unixSockets := map[int][]UnixSocket{
		42: {{Fd: "3", Device: "0x3333", PeerDevice: "0x1111"}},
		1:  {{Fd: "3", Device: "0x1111", Path: "/var/run/docker.sock"}},
		2:  {{Fd: "3", Device: "0x1111", Path: "/var/run/docker.sock"}},
	}

	connections := UnixSocketConnections(me, []*Process{me, parent, child}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "dockerd", Pid: 1},
			Protocol:  ProtocolUnix,
			Direction: DirectionOutgoing,
			Path:      "/var/run/docker.sock",
			Count:     1,
		},
	})
}

// The same fork inheritance seen from the server's side: the socket naming ours
// is held by a parent and a child alike, and the lowest PID wins there too.
//
// Which is a second thing to get right rather than the same one twice. Finding
// the process at the other end of a socket we named is an index lookup, while
// finding the processes that named a socket of ours is a scan of the whole
// listing, and it is the scan that can hand out one line per holder.
func TestUnixSocketConnections_inheritedClient(t *testing.T) {
	me := &Process{Pid: 1, Cmdline: "dockerd"}
	parent := &Process{Pid: 42, Cmdline: "compose"}
	child := &Process{Pid: 43, Cmdline: "compose"}

	unixSockets := map[int][]UnixSocket{
		42: {{Fd: "3", Device: "0x3333", PeerDevice: "0x2222"}},
		43: {{Fd: "3", Device: "0x3333", PeerDevice: "0x2222"}},
		1:  {{Fd: "4", Device: "0x2222", Path: "/var/run/docker.sock"}},
	}

	connections := UnixSocketConnections(me, []*Process{me, parent, child}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "compose", Pid: 42},
			Protocol:  ProtocolUnix,
			Direction: DirectionIncoming,
			Path:      "/var/run/docker.sock",
			Count:     1,
		},
	})
}

// A process dialing a socket of its own holds both ends, and that is one
// connection deserving one line. Both ends being ours makes us the dialer and
// the dialed at once, so there is no arrow to draw.
func TestUnixSocketConnections_selfConnection(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "dockerd"}

	unixSockets := map[int][]UnixSocket{
		42: {
			{Fd: "3", Device: "0x1111", Path: "/var/run/docker.sock"},
			{Fd: "4", Device: "0x2222", Path: "/var/run/docker.sock"},
			{Fd: "5", Device: "0x3333", PeerDevice: "0x2222"},
		},
	}

	connections := UnixSocketConnections(me, []*Process{me}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "dockerd", Pid: 42},
			Protocol:  ProtocolUnix,
			Direction: DirectionUnknown,
			Path:      "/var/run/docker.sock",
			Count:     1,
		},
	})
}

// The same self connection as reported on Linux, where the peer edge is symmetric
// and so finds it twice over from either end.
//
// Which is what makes the direction of a self connection unknown here: the two
// notes disagree about whose end carries the path, so both of them end up
// carrying one. The answer has to be the same whichever order they arrive in,
// map iteration order being what decides it.
func TestUnixSocketConnections_linuxSelfConnection(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "dockerd"}

	unixSockets := map[int][]UnixSocket{
		42: {
			{Fd: "3", Device: "0x1111", Path: "/var/run/docker.sock"},
			{Fd: "4", Device: "0x2222", PeerDevice: "0x3333", Path: "/var/run/docker.sock"},
			{Fd: "5", Device: "0x3333", PeerDevice: "0x2222"},
		},
	}

	connections := UnixSocketConnections(me, []*Process{me}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "dockerd", Pid: 42},
			Protocol:  ProtocolUnix,
			Direction: DirectionUnknown,
			Path:      "/var/run/docker.sock",
			Count:     1,
		},
	})
}

// A client that bound an address of its own before dialing carries a path just
// like the socket it dialed, and then there is no telling which of the two is the
// server. The connection is still worth a line — it is the arrow that goes
// missing, not the peer.
//
// Binding first is cheap in the abstract namespace and some D-Bus and X11 clients
// do it. Telling this apart from a server would take knowing who called listen(2),
// which neither lsof nor the netlink dump reports; see unixSocketDirection().
//
// The path reported is our own bound address rather than the server's, both ends
// having one and our own being the one preferred. Whoever is looked at gets their
// own, so the two ends of this connection describe it differently — the same
// missing fact as the direction, showing up in the other field.
func TestUnixSocketConnections_clientWithAPathOfItsOwn(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "curl"}
	server := &Process{Pid: 1, Cmdline: "dockerd"}

	unixSockets := map[int][]UnixSocket{
		42: {{Fd: "3", Device: "0x3333", PeerDevice: "0x2222", Path: "@curl-4711"}},
		1: {
			{Fd: "3", Device: "0x1111", Path: "/var/run/docker.sock"},
			{Fd: "4", Device: "0x2222", PeerDevice: "0x3333", Path: "/var/run/docker.sock"},
		},
	}

	connections := UnixSocketConnections(me, []*Process{me, server}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Name: "dockerd", Pid: 1},
			Protocol:  ProtocolUnix,
			Direction: DirectionUnknown,
			Path:      "@curl-4711",
			Count:     1,
		},
	})
}

// lsof runs after the process listing, so a peer can be a process we have no
// name for. Its PID is still worth showing.
func TestUnixSocketConnections_namelessPeer(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "curl"}

	unixSockets := map[int][]UnixSocket{
		42: {{Fd: "3", Device: "0x3333", PeerDevice: "0x2222"}},
		1:  {{Fd: "4", Device: "0x2222", Path: "/var/run/docker.sock"}},
	}

	connections := UnixSocketConnections(me, []*Process{me}, unixSockets)

	assert.SlicesEqual(t, connections, []Connection{
		{
			Peer:      Peer{Pid: 1},
			Protocol:  ProtocolUnix,
			Direction: DirectionOutgoing,
			Path:      "/var/run/docker.sock",
			Count:     1,
		},
	})
}

// A process holding no unix sockets at all, which is most of them, has nothing
// to report. Neither has one we aren't allowed to inspect, and this is what both
// look like.
func TestUnixSocketConnections_noUnixSocketsAtAll(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "curl"}

	connections := UnixSocketConnections(me, []*Process{me}, map[int][]UnixSocket{})

	assert.SlicesEqual(t, connections, []Connection(nil))
}

// A real listing should tell us enough about a real connected pair to match it,
// on either platform.
//
// Both ends are held by this very process, so the connection this finds is one
// to ourselves, over the path we made it on.
func TestUnixSocketConnections_realUnixSocket(t *testing.T) {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof not available: ", err)
	}

	// Not t.TempDir(): a unix socket path has a length limit around 100
	// characters, and a temporary directory named after this test eats into it.
	directory, err := os.MkdirTemp("", "ftop")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(directory) }()

	path := filepath.Join(directory, "probe.sock")

	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()

	client, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	accepted, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = accepted.Close() }()

	unixSocketsByPid, err := GetUnixSocketsByPid()
	if err != nil {
		t.Fatalf("listing unix sockets failed: %v", err)
	}

	me := &Process{Pid: os.Getpid(), Cmdline: "processes.test"}

	connections := UnixSocketConnections(me, []*Process{me}, unixSocketsByPid)

	found := false
	for _, connection := range connections {
		if connection.Path != path {
			continue
		}

		assert.Equal(t, connection.Protocol, ProtocolUnix)
		assert.Equal(t, connection.Peer.Pid, me.Pid)

		found = true
	}

	if !found {
		t.Fatalf("found %v, expected a connection over %s", connections, path)
	}
}
