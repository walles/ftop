package processes

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// The real lsof should tell us enough about a real connected pair to match it.
//
// Both ends are held by this very process, so the connection this finds is one
// to ourselves, over the path we made it on.
//
// macOS only, for the reason TestGetUnixSocketsByPid() gives: Linux lsof reports
// no peer for a unix socket, so there is nothing here to match on until the
// netlink collector lands.
func TestUnixSocketConnections_realUnixSocket(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("lsof only reports a unix socket's peer on macOS")
	}

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
