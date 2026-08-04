package processes

import (
	"fmt"
	"testing"

	"github.com/walles/ftop/internal/assert"
)

// A connection to somebody we can't find a process for is a connection to a
// remote host, named by its address.
func TestNetworkConnections_outgoingToRemoteHost(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {{Fd: "3", Local: "192.168.50.32:57759", Remote: "140.82.114.25:443"}},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "140.82.114.25"}, Direction: DirectionOutgoing, Port: 443, Count: 1},
	})
}

// Not listening on a port means we can't have accepted on it, so a connection
// from a port of ours that we do listen on is one somebody else dialed.
func TestNetworkConnections_incomingFromRemoteHost(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Local: "192.168.50.32:8080", Listening: true},
			{Fd: "4", Local: "192.168.50.32:8080", Remote: "1.2.3.4:33102"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Direction: DirectionIncoming, Port: 8080, Listening: true, Count: 1},
		{Peer: Peer{Name: "1.2.3.4"}, Direction: DirectionIncoming, Port: 8080, Count: 1},
	})
}

// A socket accepted by a wildcard listener reports a concrete local address,
// which never matches the "*:8082" the listener is bound to. Matching on the
// port alone is what connects the two.
func TestNetworkConnections_wildcardListener(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Local: "*:8082", Listening: true},
			{Fd: "4", Local: "192.168.50.32:8082", Remote: "1.2.3.4:48368"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Direction: DirectionIncoming, Port: 8082, Listening: true, Count: 1},
		{Peer: Peer{Name: "1.2.3.4"}, Direction: DirectionIncoming, Port: 8082, Count: 1},
	})
}

// A listener bound to one address only accepts connections to that address, so
// a connection from the same port of a different address of ours is one we
// dialed ourselves.
func TestNetworkConnections_listenerBoundToOneAddress(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Local: "127.0.0.1:8080", Listening: true},
			{Fd: "4", Local: "10.0.0.5:8080", Remote: "1.2.3.4:9999"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Direction: DirectionIncoming, Port: 8080, Listening: true, Count: 1},
		{Peer: Peer{Name: "1.2.3.4"}, Direction: DirectionOutgoing, Port: 9999, Count: 1},
	})
}

// A server that accepts a connection and hands it to a child process keeps the
// listening socket in the parent, and a socket activated server never holds one
// at all. The port is one connections arrive on either way, so what decides the
// direction is the ports being listened on anywhere on this machine rather than
// only the ones we hold ourselves.
func TestNetworkConnections_incomingViaAnotherProcessesListener(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "sshd"}
	listeningParent := &Process{Pid: 10, Cmdline: "sshd"}

	sockets := map[int][]Socket{
		42: {{Fd: "9", Local: "192.168.50.32:22", Remote: "1.2.3.4:54321"}},
		10: {{Fd: "3", Local: "*:22", Listening: true}},
	}

	connections := NetworkConnections(me, []*Process{me, listeningParent}, sockets)

	// The listening row belongs to the parent, not to us: we don't hold that
	// socket.
	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "1.2.3.4"}, Direction: DirectionIncoming, Port: 22, Count: 1},
	})
}

// Somebody else's listener bound to one address accepts connections to that
// address, whichever process ends up holding the accepted socket.
func TestNetworkConnections_incomingViaAnotherProcessesBoundListener(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "worker"}
	listeningParent := &Process{Pid: 10, Cmdline: "server"}

	sockets := map[int][]Socket{
		42: {{Fd: "9", Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"}},
		10: {{Fd: "3", Local: "127.0.0.1:8080", Listening: true}},
	}

	connections := NetworkConnections(me, []*Process{me, listeningParent}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "127.0.0.1"}, Direction: DirectionIncoming, Port: 8080, Count: 1},
	})
}

// The process at the other end of a connection is the one holding a socket with
// our own endpoints reversed. A TCP connection is its four endpoint numbers, so
// there is at most one such process.
func TestNetworkConnections_peerIsALocalProcess(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}
	curl := &Process{Pid: 999, Cmdline: "curl"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Local: "127.0.0.1:8080", Listening: true},
			{Fd: "4", Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"},
		},
		999: {{Fd: "7", Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"}},
	}

	connections := NetworkConnections(me, []*Process{me, curl}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Direction: DirectionIncoming, Port: 8080, Listening: true, Count: 1},
		{Peer: Peer{Name: "curl", Pid: 999}, Direction: DirectionIncoming, Port: 8080, Count: 1},
	})
}

// lsof runs after the process listing, so a peer can be a process that didn't
// exist yet when we listed them. We still know its PID.
func TestNetworkConnections_peerProcessWithoutAName(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42:  {{Fd: "3", Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"}},
		999: {{Fd: "7", Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"}},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Pid: 999}, Direction: DirectionOutgoing, Port: 8080, Count: 1},
	})
}

// A socket inherited across a fork belongs to parent and child alike, and the
// same connection must be reported as the same peer every time we look.
func TestNetworkConnections_peerSocketSharedByForkedProcesses(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}
	server := &Process{Pid: 200, Cmdline: "server"}
	worker := &Process{Pid: 300, Cmdline: "worker"}

	sockets := map[int][]Socket{
		42:  {{Fd: "3", Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"}},
		200: {{Fd: "7", Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"}},
		300: {{Fd: "7", Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"}},
	}

	connections := NetworkConnections(me, []*Process{me, server, worker}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "server", Pid: 200}, Direction: DirectionOutgoing, Port: 8080, Count: 1},
	})
}

// Sockets that are bound but were never connected have nothing to report. They
// must not turn into a peerless connection on some port we made up.
func TestNetworkConnections_ignoresUnconnectedSockets(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Local: "*:60000"},
			{Fd: "4", Local: "192.168.50.32:50000", Remote: "1.2.3.4:443"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "1.2.3.4"}, Direction: DirectionOutgoing, Port: 443, Count: 1},
	})
}

// Processes really do hold hundreds of connections to one and the same peer.
// Listing them one line each says nothing a count doesn't say better.
func TestNetworkConnections_aggregatesIdenticalConnections(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	var mySockets []Socket
	for i := range 3 {
		mySockets = append(mySockets, Socket{
			Fd:     fmt.Sprint(i),
			Local:  fmt.Sprintf("192.168.50.32:%d", 50000+i),
			Remote: "140.82.114.25:443",
		})
	}

	connections := NetworkConnections(me, []*Process{me}, map[int][]Socket{42: mySockets})

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "140.82.114.25"}, Direction: DirectionOutgoing, Port: 443, Count: 3},
	})
}

// Connections to different ports of the same peer are different connections,
// and aggregating them together would hide what the peer is being used for.
// With the peer and the direction tied, the lower port comes first.
func TestNetworkConnections_aggregatesPerPort(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Local: "192.168.50.32:50000", Remote: "1.2.3.4:443"},
			{Fd: "4", Local: "192.168.50.32:50001", Remote: "1.2.3.4:80"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "1.2.3.4"}, Direction: DirectionOutgoing, Port: 80, Count: 1},
		{Peer: Peer{Name: "1.2.3.4"}, Direction: DirectionOutgoing, Port: 443, Count: 1},
	})
}

// A dual stack listener is one logical listening port held open by two sockets.
// Reporting it as two, or as one with a count of two, would both be lies.
func TestNetworkConnections_dualStackListenerIsOneRow(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Local: "0.0.0.0:7000", Listening: true},
			{Fd: "4", Local: "[::]:7000", Listening: true},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Direction: DirectionIncoming, Port: 7000, Listening: true, Count: 1},
	})
}

// Some lsof versions report the same open file once per thread of the process
// holding it. One file descriptor is one socket however many times we hear
// about it.
func TestNetworkConnections_deduplicatesRepeatedFileDescriptors(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Local: "192.168.50.32:50000", Remote: "1.2.3.4:443"},
			{Fd: "3", Local: "192.168.50.32:50000", Remote: "1.2.3.4:443"},
			{Fd: "4", Local: "192.168.50.32:50001", Remote: "1.2.3.4:443"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "1.2.3.4"}, Direction: DirectionOutgoing, Port: 443, Count: 2},
	})
}

// A process talking to itself holds both ends of the connection. That is one
// connection and gets one line, even though we can see it from both sides.
func TestNetworkConnections_selfConnectionIsShownOnce(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Local: "127.0.0.1:8080", Listening: true},
			{Fd: "4", Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"},
			{Fd: "5", Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	// Of the two sockets, the one whose local endpoint sorts first is the one
	// we keep, and that is the dialing end here.
	assert.SlicesEqual(t, connections, []Connection{
		{Direction: DirectionIncoming, Port: 8080, Listening: true, Count: 1},
		{Peer: Peer{Name: "picked", Pid: 42}, Direction: DirectionOutgoing, Port: 8080, Count: 1},
	})
}

// Listening first, then incoming, then outgoing, so that each way of drawing a
// line stays in one block. Within a block it's peer name, then PID.
func TestNetworkConnections_ordering(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}
	laterZebra := &Process{Pid: 300, Cmdline: "zebra"}
	earlierZebra := &Process{Pid: 200, Cmdline: "zebra"}
	aardvark := &Process{Pid: 400, Cmdline: "aardvark"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Local: "127.0.0.1:8080", Listening: true},
			{Fd: "4", Local: "127.0.0.1:8080", Remote: "127.0.0.1:1300"},
			{Fd: "5", Local: "127.0.0.1:8080", Remote: "127.0.0.1:1200"},
			{Fd: "6", Local: "127.0.0.1:8080", Remote: "127.0.0.1:1400"},
			{Fd: "7", Local: "127.0.0.1:60000", Remote: "127.0.0.1:9999"},
		},
		300: {{Fd: "3", Local: "127.0.0.1:1300", Remote: "127.0.0.1:8080"}},
		200: {{Fd: "3", Local: "127.0.0.1:1200", Remote: "127.0.0.1:8080"}},
		400: {{Fd: "3", Local: "127.0.0.1:1400", Remote: "127.0.0.1:8080"}},
	}

	allProcesses := []*Process{me, laterZebra, earlierZebra, aardvark}
	connections := NetworkConnections(me, allProcesses, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Direction: DirectionIncoming, Port: 8080, Listening: true, Count: 1},
		{Peer: Peer{Name: "aardvark", Pid: 400}, Direction: DirectionIncoming, Port: 8080, Count: 1},
		{Peer: Peer{Name: "zebra", Pid: 200}, Direction: DirectionIncoming, Port: 8080, Count: 1},
		{Peer: Peer{Name: "zebra", Pid: 300}, Direction: DirectionIncoming, Port: 8080, Count: 1},
		{Peer: Peer{Name: "127.0.0.1"}, Direction: DirectionOutgoing, Port: 9999, Count: 1},
	})
}

// IPv6 endpoints are bracketed to keep the address apart from the port. The
// address on its own needs no brackets, and reverse DNS won't take them.
func TestNetworkConnections_ipv6(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Local: "[::1]:8081", Listening: true},
			{Fd: "4", Local: "[::1]:8081", Remote: "[::1]:46208"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Direction: DirectionIncoming, Port: 8081, Listening: true, Count: 1},
		{Peer: Peer{Name: "::1"}, Direction: DirectionIncoming, Port: 8081, Count: 1},
	})
}

// Other processes' connections are none of our business, and lsof leaves out
// the processes it isn't allowed to inspect anyway.
func TestNetworkConnections_noSocketsOfOurOwn(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		999: {{Fd: "7", Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"}},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.Equal(t, len(connections), 0)
}
