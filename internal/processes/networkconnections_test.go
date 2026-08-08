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
		42: {{Fd: "3", Protocol: ProtocolTcp, Local: "192.168.50.32:57759", Remote: "140.82.114.25:443"}},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "140.82.114.25"}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 443, Count: 1},
	})
}

// Not listening on a port means we can't have accepted on it, so a connection
// from a port of ours that we do listen on is one somebody else dialed.
func TestNetworkConnections_incomingFromRemoteHost(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Protocol: ProtocolTcp, Local: "192.168.50.32:8080", Listening: true},
			{Fd: "4", Protocol: ProtocolTcp, Local: "192.168.50.32:8080", Remote: "1.2.3.4:33102"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Listening: true, Count: 1},
		{Peer: Peer{Name: "1.2.3.4"}, Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Count: 1},
	})
}

// A socket accepted by a wildcard listener reports a concrete local address,
// which never matches the "*:8082" the listener is bound to. Matching on the
// port alone is what connects the two.
func TestNetworkConnections_wildcardListener(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Protocol: ProtocolTcp, Local: "*:8082", Listening: true},
			{Fd: "4", Protocol: ProtocolTcp, Local: "192.168.50.32:8082", Remote: "1.2.3.4:48368"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8082, Listening: true, Count: 1},
		{Peer: Peer{Name: "1.2.3.4"}, Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8082, Count: 1},
	})
}

// A listener bound to one address only accepts connections to that address, so
// a connection from the same port of a different address of ours is one we
// dialed ourselves.
func TestNetworkConnections_listenerBoundToOneAddress(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Listening: true},
			{Fd: "4", Protocol: ProtocolTcp, Local: "10.0.0.5:8080", Remote: "1.2.3.4:9999"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Listening: true, Count: 1},
		{Peer: Peer{Name: "1.2.3.4"}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 9999, Count: 1},
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
		42: {{Fd: "9", Protocol: ProtocolTcp, Local: "192.168.50.32:22", Remote: "1.2.3.4:54321"}},
		10: {{Fd: "3", Protocol: ProtocolTcp, Local: "*:22", Listening: true}},
	}

	connections := NetworkConnections(me, []*Process{me, listeningParent}, sockets)

	// The listening row belongs to the parent, not to us: we don't hold that
	// socket.
	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "1.2.3.4"}, Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 22, Count: 1},
	})
}

// Somebody else's listener bound to one address accepts connections to that
// address, whichever process ends up holding the accepted socket.
func TestNetworkConnections_incomingViaAnotherProcessesBoundListener(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "worker"}
	listeningParent := &Process{Pid: 10, Cmdline: "server"}

	sockets := map[int][]Socket{
		42: {{Fd: "9", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"}},
		10: {{Fd: "3", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Listening: true}},
	}

	connections := NetworkConnections(me, []*Process{me, listeningParent}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "127.0.0.1"}, Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Count: 1},
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
			{Fd: "3", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Listening: true},
			{Fd: "4", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"},
		},
		999: {{Fd: "7", Protocol: ProtocolTcp, Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"}},
	}

	connections := NetworkConnections(me, []*Process{me, curl}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Listening: true, Count: 1},
		{Peer: Peer{Name: "curl", Pid: 999}, Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Count: 1},
	})
}

// lsof runs after the process listing, so a peer can be a process that didn't
// exist yet when we listed them. We still know its PID.
func TestNetworkConnections_peerProcessWithoutAName(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42:  {{Fd: "3", Protocol: ProtocolTcp, Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"}},
		999: {{Fd: "7", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"}},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Pid: 999}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 8080, Count: 1},
	})
}

// A socket inherited across a fork belongs to parent and child alike, and the
// same connection must be reported as the same peer every time we look.
func TestNetworkConnections_peerSocketSharedByForkedProcesses(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}
	server := &Process{Pid: 200, Cmdline: "server"}
	worker := &Process{Pid: 300, Cmdline: "worker"}

	sockets := map[int][]Socket{
		42:  {{Fd: "3", Protocol: ProtocolTcp, Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"}},
		200: {{Fd: "7", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"}},
		300: {{Fd: "7", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"}},
	}

	connections := NetworkConnections(me, []*Process{me, server, worker}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "server", Pid: 200}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 8080, Count: 1},
	})
}

// Sockets that are bound but were never connected have nothing to report. They
// must not turn into a peerless connection on some port we made up.
func TestNetworkConnections_ignoresUnconnectedSockets(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Protocol: ProtocolTcp, Local: "*:60000"},
			{Fd: "4", Protocol: ProtocolTcp, Local: "192.168.50.32:50000", Remote: "1.2.3.4:443"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "1.2.3.4"}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 443, Count: 1},
	})
}

// Processes really do hold hundreds of connections to one and the same peer.
// Listing them one line each says nothing a count doesn't say better.
func TestNetworkConnections_aggregatesIdenticalConnections(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	var mySockets []Socket
	for i := range 3 {
		mySockets = append(mySockets, Socket{
			Fd:       fmt.Sprint(i),
			Protocol: ProtocolTcp,
			Local:    fmt.Sprintf("192.168.50.32:%d", 50000+i),
			Remote:   "140.82.114.25:443",
		})
	}

	connections := NetworkConnections(me, []*Process{me}, map[int][]Socket{42: mySockets})

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "140.82.114.25"}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 443, Count: 3},
	})
}

// Connections to different ports of the same peer are different connections,
// and aggregating them together would hide what the peer is being used for.
// With the peer and the direction tied, the lower port comes first.
func TestNetworkConnections_aggregatesPerPort(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Protocol: ProtocolTcp, Local: "192.168.50.32:50000", Remote: "1.2.3.4:443"},
			{Fd: "4", Protocol: ProtocolTcp, Local: "192.168.50.32:50001", Remote: "1.2.3.4:80"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "1.2.3.4"}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 80, Count: 1},
		{Peer: Peer{Name: "1.2.3.4"}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 443, Count: 1},
	})
}

// A dual stack listener is one logical listening port held open by two sockets.
// Reporting it as two, or as one with a count of two, would both be lies.
func TestNetworkConnections_dualStackListenerIsOneRow(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Protocol: ProtocolTcp, Local: "0.0.0.0:7000", Listening: true},
			{Fd: "4", Protocol: ProtocolTcp, Local: "[::]:7000", Listening: true},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 7000, Listening: true, Count: 1},
	})
}

// Some lsof versions report the same open file once per thread of the process
// holding it, repeating the whole record. However many times we hear about a
// socket, it is one socket, while a connection to another port of the same peer
// is a second connection rather than another copy of the first.
func TestNetworkConnections_deduplicatesTheSameSocketReportedTwice(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Protocol: ProtocolTcp, Local: "192.168.50.32:50000", Remote: "1.2.3.4:443"},
			{Fd: "3", Protocol: ProtocolTcp, Local: "192.168.50.32:50000", Remote: "1.2.3.4:443"},
			{Fd: "4", Protocol: ProtocolTcp, Local: "192.168.50.32:50001", Remote: "1.2.3.4:443"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "1.2.3.4"}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 443, Count: 2},
	})
}

// One socket handed to a process on several file descriptors, by dup(2) or by
// being inherited as stdin, stdout and stderr from a socket activated server, is
// reported once per descriptor. A TCP connection is its four endpoint numbers, so
// two sockets of ours carrying the same four are one connection, and counting
// them separately would claim connections that don't exist.
func TestNetworkConnections_deduplicatesOneSocketOnSeveralFileDescriptors(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Protocol: ProtocolTcp, Local: "192.168.50.32:50000", Remote: "140.82.114.25:443"},
			{Fd: "4", Protocol: ProtocolTcp, Local: "192.168.50.32:50000", Remote: "140.82.114.25:443"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "140.82.114.25"}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 443, Count: 1},
	})
}

// A program that listens on a port and also dials out from it, the way a peer to
// peer client does with SO_REUSEADDR, holds a listener and a bound but not yet
// connected socket on one and the same address. Those two carry the same protocol,
// the same local address and no remote at all, and only their listening state
// tells them apart. Taking them for one socket costs the listening row.
func TestNetworkConnections_listenerAlongsideABoundSocketOnItsAddress(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			// The bound socket is listed first, so a listener mistaken for a repeat
			// of it is the one that would be dropped.
			{Fd: "3", Protocol: ProtocolTcp, Local: "127.0.0.1:8080"},
			{Fd: "4", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Listening: true},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Listening: true, Count: 1},
	})
}

// A process talking to itself holds both ends of the connection. That is one
// connection and gets one line, even though we can see it from both sides.
func TestNetworkConnections_selfConnectionIsShownOnce(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Listening: true},
			{Fd: "4", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"},
			{Fd: "5", Protocol: ProtocolTcp, Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	// Of the two sockets, the one whose local endpoint sorts first is the one
	// we keep, and that is the dialing end here.
	assert.SlicesEqual(t, connections, []Connection{
		{Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Listening: true, Count: 1},
		{Peer: Peer{Name: "picked", Pid: 42}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 8080, Count: 1},
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
			{Fd: "3", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Listening: true},
			{Fd: "4", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:1300"},
			{Fd: "5", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:1200"},
			{Fd: "6", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:1400"},
			{Fd: "7", Protocol: ProtocolTcp, Local: "127.0.0.1:60000", Remote: "127.0.0.1:9999"},
		},
		300: {{Fd: "3", Protocol: ProtocolTcp, Local: "127.0.0.1:1300", Remote: "127.0.0.1:8080"}},
		200: {{Fd: "3", Protocol: ProtocolTcp, Local: "127.0.0.1:1200", Remote: "127.0.0.1:8080"}},
		400: {{Fd: "3", Protocol: ProtocolTcp, Local: "127.0.0.1:1400", Remote: "127.0.0.1:8080"}},
	}

	allProcesses := []*Process{me, laterZebra, earlierZebra, aardvark}
	connections := NetworkConnections(me, allProcesses, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Listening: true, Count: 1},
		{Peer: Peer{Name: "aardvark", Pid: 400}, Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Count: 1},
		{Peer: Peer{Name: "zebra", Pid: 200}, Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Count: 1},
		{Peer: Peer{Name: "zebra", Pid: 300}, Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Count: 1},
		{Peer: Peer{Name: "127.0.0.1"}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 9999, Count: 1},
	})
}

// IPv6 endpoints are bracketed to keep the address apart from the port. The
// address on its own needs no brackets, and reverse DNS won't take them.
func TestNetworkConnections_ipv6(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Protocol: ProtocolTcp, Local: "[::1]:8081", Listening: true},
			{Fd: "4", Protocol: ProtocolTcp, Local: "[::1]:8081", Remote: "[::1]:46208"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8081, Listening: true, Count: 1},
		{Peer: Peer{Name: "::1"}, Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8081, Count: 1},
	})
}

// UDP carries no state for lsof to report, and a UDP socket is bound as soon as
// it sends, so there is no listening port anywhere on the machine to compare ours
// against and no telling which end started the conversation. Of the two ports the
// peer's is the one reported.
func TestNetworkConnections_udpDirectionIsUnknown(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {{Fd: "3", Protocol: ProtocolUdp, Local: "192.168.50.32:51293", Remote: "8.8.8.8:53"}},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "8.8.8.8"}, Protocol: ProtocolUdp, Direction: DirectionUnknown, Port: 53, Count: 1},
	})
}

// Listening on a TCP port says nothing about the same port number over UDP, so a
// UDP connection from a port we do listen on over TCP is still one nobody can tell
// the direction of. Both halves of the listen set are TCP's alone: the ports of
// the wildcard listeners, and the fully spelled out endpoints of the rest.
func TestNetworkConnections_udpIgnoresTcpListeners(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Protocol: ProtocolTcp, Local: "*:8080", Listening: true},
			{Fd: "4", Protocol: ProtocolUdp, Local: "192.168.50.32:8080", Remote: "1.2.3.4:9999"},
			{Fd: "5", Protocol: ProtocolTcp, Local: "127.0.0.1:9090", Listening: true},
			{Fd: "6", Protocol: ProtocolUdp, Local: "127.0.0.1:9090", Remote: "1.2.3.4:9998"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Listening: true, Count: 1},
		{Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 9090, Listening: true, Count: 1},
		{Peer: Peer{Name: "1.2.3.4"}, Protocol: ProtocolUdp, Direction: DirectionUnknown, Port: 9998, Count: 1},
		{Peer: Peer{Name: "1.2.3.4"}, Protocol: ProtocolUdp, Direction: DirectionUnknown, Port: 9999, Count: 1},
	})
}

// A process can hold both ends of a TCP conversation and, separately, a UDP socket
// whose endpoints happen to be the reverse of it. The UDP socket is not the far end
// of the TCP connection, so neither of them gets dropped as a duplicate of the
// other, and neither is named as the other's peer.
func TestNetworkConnections_udpIsNotTheFarEndOfATcpConnection(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Protocol: ProtocolTcp, Local: "127.0.0.1:1000", Remote: "127.0.0.1:2000"},
			{Fd: "4", Protocol: ProtocolUdp, Local: "127.0.0.1:2000", Remote: "127.0.0.1:1000"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "127.0.0.1"}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 2000, Count: 1},
		{Peer: Peer{Name: "127.0.0.1"}, Protocol: ProtocolUdp, Direction: DirectionUnknown, Port: 1000, Count: 1},
	})
}

// The peer of a UDP connection is found exactly the way a TCP peer is, by looking
// for the process holding our own endpoints the other way around.
func TestNetworkConnections_udpPeerIsALocalProcess(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}
	resolver := &Process{Pid: 999, Cmdline: "dnsmasq"}

	sockets := map[int][]Socket{
		42:  {{Fd: "3", Protocol: ProtocolUdp, Local: "127.0.0.1:51293", Remote: "127.0.0.1:53"}},
		999: {{Fd: "7", Protocol: ProtocolUdp, Local: "127.0.0.1:53", Remote: "127.0.0.1:51293"}},
	}

	connections := NetworkConnections(me, []*Process{me, resolver}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "dnsmasq", Pid: 999}, Protocol: ProtocolUdp, Direction: DirectionUnknown, Port: 53, Count: 1},
	})
}

// A TCP and a UDP connection can carry the very same four endpoint numbers while
// having nothing to do with each other. Neither may be taken for the other's peer,
// and together they are two connections rather than one counted twice.
func TestNetworkConnections_udpAndTcpAreToldApart(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}
	tcpServer := &Process{Pid: 200, Cmdline: "tcpserver"}
	udpServer := &Process{Pid: 300, Cmdline: "udpserver"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Protocol: ProtocolTcp, Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"},
			{Fd: "4", Protocol: ProtocolUdp, Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"},
		},
		200: {{Fd: "7", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"}},
		300: {{Fd: "7", Protocol: ProtocolUdp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"}},
	}

	connections := NetworkConnections(me, []*Process{me, tcpServer, udpServer}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "tcpserver", Pid: 200}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 8080, Count: 1},
		{Peer: Peer{Name: "udpserver", Pid: 300}, Protocol: ProtocolUdp, Direction: DirectionUnknown, Port: 8080, Count: 1},
	})
}

// Most UDP sockets are bound without ever being connected, and UDP has no
// listening state to tell a server's socket from the ephemeral source port of
// something that merely sent a packet. Rather than call them all one or all the
// other, they are left out — so a process serving UDP and nothing else, holding
// only "*:53", gets no line at all.
func TestNetworkConnections_ignoresBoundUdpSockets(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Protocol: ProtocolUdp, Local: "*:53"},
			{Fd: "4", Protocol: ProtocolUdp, Local: "*:*"},
			{Fd: "5", Protocol: ProtocolUdp, Local: "127.0.0.1:51293", Remote: "127.0.0.1:53"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "127.0.0.1"}, Protocol: ProtocolUdp, Direction: DirectionUnknown, Port: 53, Count: 1},
	})
}

// Every way of drawing a connection keeps to its own block of the listing:
// listening ports, then who dialed in, then who we dialed, then the ones nobody
// can tell the direction of.
func TestNetworkConnections_undeterminedDirectionSortsLast(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		42: {
			{Fd: "3", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Listening: true},
			{Fd: "4", Protocol: ProtocolUdp, Local: "127.0.0.1:51293", Remote: "1.2.3.4:53"},
			{Fd: "5", Protocol: ProtocolTcp, Local: "127.0.0.1:8080", Remote: "1.2.3.4:33102"},
			{Fd: "6", Protocol: ProtocolTcp, Local: "127.0.0.1:60000", Remote: "1.2.3.4:443"},
		},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.SlicesEqual(t, connections, []Connection{
		{Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Listening: true, Count: 1},
		{Peer: Peer{Name: "1.2.3.4"}, Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Count: 1},
		{Peer: Peer{Name: "1.2.3.4"}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 443, Count: 1},
		{Peer: Peer{Name: "1.2.3.4"}, Protocol: ProtocolUdp, Direction: DirectionUnknown, Port: 53, Count: 1},
	})
}

// Other processes' connections are none of our business, and lsof leaves out
// the processes it isn't allowed to inspect anyway.
func TestNetworkConnections_noSocketsOfOurOwn(t *testing.T) {
	me := &Process{Pid: 42, Cmdline: "picked"}

	sockets := map[int][]Socket{
		999: {{Fd: "7", Protocol: ProtocolTcp, Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"}},
	}

	connections := NetworkConnections(me, []*Process{me}, sockets)

	assert.Equal(t, len(connections), 0)
}

// A section showing more than one kind of connection sorts them together, and
// the protocols must not interleave: the description column is what a reader
// scans, and a block of pipes broken up by a socket line reads as noise.
//
// The peer names here are chosen so that sorting by name alone would interleave
// them, which is what makes this test say anything.
func TestSortConnections_keepsProtocolsApart(t *testing.T) {
	connections := []Connection{
		{Peer: Peer{Name: "alpha", Pid: 1}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 443, Count: 1},
		{Peer: Peer{Name: "beta", Pid: 2}, Protocol: ProtocolPipe, Direction: DirectionOutgoing, Count: 1},
		{Peer: Peer{Name: "gamma", Pid: 3}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 22, Count: 1},
		{Peer: Peer{Name: "delta", Pid: 4}, Protocol: ProtocolPipe, Direction: DirectionOutgoing, Count: 1},
	}

	SortConnections(connections)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "beta", Pid: 2}, Protocol: ProtocolPipe, Direction: DirectionOutgoing, Count: 1},
		{Peer: Peer{Name: "delta", Pid: 4}, Protocol: ProtocolPipe, Direction: DirectionOutgoing, Count: 1},
		{Peer: Peer{Name: "alpha", Pid: 1}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 443, Count: 1},
		{Peer: Peer{Name: "gamma", Pid: 3}, Protocol: ProtocolTcp, Direction: DirectionOutgoing, Port: 22, Count: 1},
	})
}

// Which way a connection is drawn decides its block, and that outranks the
// protocol: an incoming pipe belongs with the incoming sockets rather than with
// the outgoing pipes.
func TestSortConnections_directionOutranksProtocol(t *testing.T) {
	connections := []Connection{
		{Peer: Peer{Name: "alpha", Pid: 1}, Protocol: ProtocolPipe, Direction: DirectionOutgoing, Count: 1},
		{Peer: Peer{Name: "beta", Pid: 2}, Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Count: 1},
		{Peer: Peer{Name: "gamma", Pid: 3}, Protocol: ProtocolPipe, Direction: DirectionIncoming, Count: 1},
	}

	SortConnections(connections)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "gamma", Pid: 3}, Protocol: ProtocolPipe, Direction: DirectionIncoming, Count: 1},
		{Peer: Peer{Name: "beta", Pid: 2}, Protocol: ProtocolTcp, Direction: DirectionIncoming, Port: 8080, Count: 1},
		{Peer: Peer{Name: "alpha", Pid: 1}, Protocol: ProtocolPipe, Direction: DirectionOutgoing, Count: 1},
	})
}
