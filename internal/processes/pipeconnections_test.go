package processes

import (
	"os"
	"os/exec"
	"testing"

	"github.com/walles/ftop/internal/assert"
)

// macOS names each end of a pipe by its peer's kernel address, and that address
// is what the peer's device field says. Which end writes it won't say, so
// neither can we.
func TestPipeConnections_macOsPair(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "grep"}
	peer := &Process{Pid: 5678, Cmdline: "sort"}

	pipeEnds := map[int][]PipeEnd{
		1234: {{Fd: "1", Device: "0xaaaa", PeerDevice: "0xbbbb"}},
		5678: {{Fd: "0", Device: "0xbbbb", PeerDevice: "0xaaaa"}},
	}

	connections := PipeConnections(me, []*Process{me, peer}, pipeEnds)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "sort", Pid: 5678}, Protocol: ProtocolPipe, Direction: DirectionUnknown, Count: 1},
	})
}

// Linux names both ends of a pipe by the pipe's inode and tells them apart by
// their access modes. Data flows from the writer to the reader, so the writing
// end is the outgoing one.
func TestPipeConnections_linuxWriteEnd(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "grep"}
	peer := &Process{Pid: 5678, Cmdline: "sort"}

	pipeEnds := map[int][]PipeEnd{
		1234: {{Fd: "1", Access: PipeAccessWrite, Inode: "16466"}},
		5678: {{Fd: "0", Access: PipeAccessRead, Inode: "16466"}},
	}

	connections := PipeConnections(me, []*Process{me, peer}, pipeEnds)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "sort", Pid: 5678}, Protocol: ProtocolPipe, Direction: DirectionOutgoing, Count: 1},
	})
}

// The reading end of the same pipe, seen from the other process: the same pipe,
// pointing the other way.
func TestPipeConnections_linuxReadEnd(t *testing.T) {
	writer := &Process{Pid: 1234, Cmdline: "grep"}
	me := &Process{Pid: 5678, Cmdline: "sort"}

	pipeEnds := map[int][]PipeEnd{
		1234: {{Fd: "1", Access: PipeAccessWrite, Inode: "16466"}},
		5678: {{Fd: "0", Access: PipeAccessRead, Inode: "16466"}},
	}

	connections := PipeConnections(me, []*Process{writer, me}, pipeEnds)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "grep", Pid: 1234}, Protocol: ProtocolPipe, Direction: DirectionIncoming, Count: 1},
	})
}

// A named FIFO is matched the way a Linux pipe is, by inode and opposing access
// modes, on either platform.
func TestPipeConnections_namedFifo(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "producer"}
	peer := &Process{Pid: 5678, Cmdline: "consumer"}

	pipeEnds := map[int][]PipeEnd{
		1234: {{Fd: "3", Access: PipeAccessWrite, Inode: "82144503"}},
		5678: {{Fd: "3", Access: PipeAccessRead, Inode: "82144503"}},
	}

	connections := PipeConnections(me, []*Process{me, peer}, pipeEnds)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "consumer", Pid: 5678}, Protocol: ProtocolPipe, Direction: DirectionOutgoing, Count: 1},
	})
}

// Two processes writing into one pipe are not talking to each other, however
// much inode they have in common. Only opposing access modes make a pair.
func TestPipeConnections_twoWritersAreNotPeers(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "grep"}
	fellowWriter := &Process{Pid: 1235, Cmdline: "sed"}
	reader := &Process{Pid: 5678, Cmdline: "sort"}

	pipeEnds := map[int][]PipeEnd{
		1234: {{Fd: "1", Access: PipeAccessWrite, Inode: "16466"}},
		1235: {{Fd: "1", Access: PipeAccessWrite, Inode: "16466"}},
		5678: {{Fd: "0", Access: PipeAccessRead, Inode: "16466"}},
	}

	connections := PipeConnections(me, []*Process{me, fellowWriter, reader}, pipeEnds)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "sort", Pid: 5678}, Protocol: ProtocolPipe, Direction: DirectionOutgoing, Count: 1},
	})
}

// One pipe end open on several file descriptors, by dup(2) or by being inherited
// as both stdout and stderr, is one pipe end. Counting the descriptors would
// report one pipe as several.
func TestPipeConnections_duplicatedDescriptors(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "grep"}
	peer := &Process{Pid: 5678, Cmdline: "sort"}

	pipeEnds := map[int][]PipeEnd{
		1234: {
			{Fd: "1", Access: PipeAccessWrite, Inode: "16466"},
			{Fd: "2", Access: PipeAccessWrite, Inode: "16466"},
		},
		5678: {
			{Fd: "0", Access: PipeAccessRead, Inode: "16466"},
			{Fd: "3", Access: PipeAccessRead, Inode: "16466"},
		},
	}

	connections := PipeConnections(me, []*Process{me, peer}, pipeEnds)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "sort", Pid: 5678}, Protocol: ProtocolPipe, Direction: DirectionOutgoing, Count: 1},
	})
}

// The same pipe end on two descriptors, the way macOS spells it: one device,
// and one pipe. The Linux sibling of this test tells the ends apart by their
// access modes, which macOS doesn't report, so what identifies an end here is
// its device alone.
func TestPipeConnections_duplicatedDescriptorsOnMacOs(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "grep"}
	peer := &Process{Pid: 5678, Cmdline: "sort"}

	pipeEnds := map[int][]PipeEnd{
		1234: {
			{Fd: "1", Device: "0xaaaa", PeerDevice: "0xbbbb"},
			{Fd: "2", Device: "0xaaaa", PeerDevice: "0xbbbb"},
		},
		5678: {
			{Fd: "0", Device: "0xbbbb", PeerDevice: "0xaaaa"},
			{Fd: "3", Device: "0xbbbb", PeerDevice: "0xaaaa"},
		},
	}

	connections := PipeConnections(me, []*Process{me, peer}, pipeEnds)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "sort", Pid: 5678}, Protocol: ProtocolPipe, Direction: DirectionUnknown, Count: 1},
	})
}

// Several pipes to the same process are several pipes, and get a count rather
// than a line each.
func TestPipeConnections_severalPipesToOnePeer(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "grep"}
	peer := &Process{Pid: 5678, Cmdline: "sort"}

	pipeEnds := map[int][]PipeEnd{
		1234: {
			{Fd: "1", Access: PipeAccessWrite, Inode: "16466"},
			{Fd: "4", Access: PipeAccessWrite, Inode: "16467"},
		},
		5678: {
			{Fd: "0", Access: PipeAccessRead, Inode: "16466"},
			{Fd: "3", Access: PipeAccessRead, Inode: "16467"},
		},
	}

	connections := PipeConnections(me, []*Process{me, peer}, pipeEnds)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "sort", Pid: 5678}, Protocol: ProtocolPipe, Direction: DirectionOutgoing, Count: 2},
	})
}

// A pipe has as many ends as anybody cares to fork, unlike a socket, which has
// exactly two. Everybody holding a reading end of ours can read what we write,
// so they all get a line.
func TestPipeConnections_severalReaders(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "grep"}
	parent := &Process{Pid: 5678, Cmdline: "sort"}
	child := &Process{Pid: 5679, Cmdline: "sort"}

	pipeEnds := map[int][]PipeEnd{
		1234: {{Fd: "1", Access: PipeAccessWrite, Inode: "16466"}},
		5678: {{Fd: "0", Access: PipeAccessRead, Inode: "16466"}},
		5679: {{Fd: "0", Access: PipeAccessRead, Inode: "16466"}},
	}

	connections := PipeConnections(me, []*Process{me, parent, child}, pipeEnds)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "sort", Pid: 5678}, Protocol: ProtocolPipe, Direction: DirectionOutgoing, Count: 1},
		{Peer: Peer{Name: "sort", Pid: 5679}, Protocol: ProtocolPipe, Direction: DirectionOutgoing, Count: 1},
	})
}

// The reader's view of the pipe two processes write into: two writers, so two
// lines, one per process it can receive from.
func TestPipeConnections_severalWriters(t *testing.T) {
	writer := &Process{Pid: 1234, Cmdline: "grep"}
	fellowWriter := &Process{Pid: 1235, Cmdline: "sed"}
	me := &Process{Pid: 5678, Cmdline: "sort"}

	pipeEnds := map[int][]PipeEnd{
		1234: {{Fd: "1", Access: PipeAccessWrite, Inode: "16466"}},
		1235: {{Fd: "1", Access: PipeAccessWrite, Inode: "16466"}},
		5678: {{Fd: "0", Access: PipeAccessRead, Inode: "16466"}},
	}

	connections := PipeConnections(me, []*Process{writer, fellowWriter, me}, pipeEnds)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "grep", Pid: 1234}, Protocol: ProtocolPipe, Direction: DirectionIncoming, Count: 1},
		{Peer: Peer{Name: "sed", Pid: 1235}, Protocol: ProtocolPipe, Direction: DirectionIncoming, Count: 1},
	})
}

// An end opened for reading and writing both, which is what "exec 3<>fifo"
// gets you, can send to a reader as well as any writer can.
//
// Which way the data goes it can't say, so the arrow points both ways.
func TestPipeConnections_readWriteEndPairsWithAReader(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "shell"}
	peer := &Process{Pid: 5678, Cmdline: "consumer"}

	pipeEnds := map[int][]PipeEnd{
		1234: {{Fd: "3", Access: PipeAccessReadWrite, Inode: "82144503"}},
		5678: {{Fd: "0", Access: PipeAccessRead, Inode: "82144503"}},
	}

	connections := PipeConnections(me, []*Process{me, peer}, pipeEnds)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "consumer", Pid: 5678}, Protocol: ProtocolPipe, Direction: DirectionUnknown, Count: 1},
	})
}

// Two processes with a FIFO open for reading and writing both can each send to
// the other, so they are peers however identical their access modes are. This is
// the one case where two ends spelled the same way do make a pair.
func TestPipeConnections_twoReadWriteEndsArePeers(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "shell"}
	peer := &Process{Pid: 5678, Cmdline: "othershell"}

	pipeEnds := map[int][]PipeEnd{
		1234: {{Fd: "3", Access: PipeAccessReadWrite, Inode: "82144503"}},
		5678: {{Fd: "3", Access: PipeAccessReadWrite, Inode: "82144503"}},
	}

	connections := PipeConnections(me, []*Process{me, peer}, pipeEnds)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "othershell", Pid: 5678}, Protocol: ProtocolPipe, Direction: DirectionUnknown, Count: 1},
	})
}

// A pipe whose other end we can't see tells us nothing: it may be held by a
// process we aren't allowed to inspect, or by nobody at all, and there is no
// telling those apart.
func TestPipeConnections_peerNotFound(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "grep"}

	pipeEnds := map[int][]PipeEnd{
		1234: {
			{Fd: "1", Access: PipeAccessWrite, Inode: "16466"},
			{Fd: "4", Device: "0xaaaa", PeerDevice: "0xbbbb"},
			// The way lsof reports a pipe with nobody at the other end
			{Fd: "5", Device: "0xcccc"},
		},
	}

	connections := PipeConnections(me, []*Process{me}, pipeEnds)

	assert.Equal(t, len(connections), 0)
}

// A process holding both ends of a pipe holds one pipe, and one pipe is one
// line. The writing end is the one it gets reported by, data flowing from there.
func TestPipeConnections_selfPipeWithAccessModes(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "grep"}

	pipeEnds := map[int][]PipeEnd{
		1234: {
			{Fd: "3", Access: PipeAccessRead, Inode: "16466"},
			{Fd: "4", Access: PipeAccessWrite, Inode: "16466"},
		},
	}

	connections := PipeConnections(me, []*Process{me}, pipeEnds)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "grep", Pid: 1234}, Protocol: ProtocolPipe, Direction: DirectionOutgoing, Count: 1},
	})
}

// The same pipe on macOS, where lsof won't say which end writes. Still one pipe
// and still one line, reported by whichever end this picks as long as it picks
// the same one every time.
func TestPipeConnections_selfPipeWithoutAccessModes(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "grep"}

	pipeEnds := map[int][]PipeEnd{
		1234: {
			{Fd: "3", Device: "0xaaaa", PeerDevice: "0xbbbb"},
			{Fd: "4", Device: "0xbbbb", PeerDevice: "0xaaaa"},
		},
	}

	connections := PipeConnections(me, []*Process{me}, pipeEnds)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Name: "grep", Pid: 1234}, Protocol: ProtocolPipe, Direction: DirectionUnknown, Count: 1},
	})
}

// lsof runs after the process listing, so a peer can be a process we have no
// name for. Its PID is still worth reporting.
func TestPipeConnections_namelessPeer(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "grep"}

	pipeEnds := map[int][]PipeEnd{
		1234: {{Fd: "1", Access: PipeAccessWrite, Inode: "16466"}},
		5678: {{Fd: "0", Access: PipeAccessRead, Inode: "16466"}},
	}

	connections := PipeConnections(me, []*Process{me}, pipeEnds)

	assert.SlicesEqual(t, connections, []Connection{
		{Peer: Peer{Pid: 5678}, Protocol: ProtocolPipe, Direction: DirectionOutgoing, Count: 1},
	})
}

// A process we hold no pipe ends of, because it has none or because we aren't
// allowed to inspect it.
func TestPipeConnections_noPipesAtAll(t *testing.T) {
	me := &Process{Pid: 1234, Cmdline: "grep"}

	connections := PipeConnections(me, []*Process{me}, map[int][]PipeEnd{})

	assert.Equal(t, len(connections), 0)
}

// A pipe we made ourselves, matched against itself through the real lsof. Both
// of its ends are ours, so this is the self-pipe case, and it is what says that
// the matching works on whatever platform and lsof version this is.
func TestPipeConnections_realPipe(t *testing.T) {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof not available: ", err)
	}

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
	}()

	pipeEndsByPid, err := GetPipeEndsByPid()
	if err != nil {
		t.Fatalf("listing pipes failed: %v", err)
	}

	me := &Process{Pid: os.Getpid(), Cmdline: "pipes.test"}

	foundOurselves := false
	for _, connection := range PipeConnections(me, []*Process{me}, pipeEndsByPid) {
		if connection.Peer.Pid != me.Pid {
			// The pipes the test runner talks to us over
			continue
		}

		// No assertion on the count: it aggregates distinct pipes to the same
		// peer, and anything else in this binary holding both ends of a pipe of
		// its own would add to it. What the count has to get right is covered by
		// the tests above, on data we control.
		assert.Equal(t, connection.Protocol, ProtocolPipe)

		foundOurselves = true
	}

	assert.Equal(t, foundOurselves, true)
}
