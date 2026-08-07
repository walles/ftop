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

// A client connected to a server over a path, which is the shape lsof reports
// for every unix domain socket that was made over the file system.
//
// The server holds two sockets on that one path: the listener it accepts on,
// and the socket that came of accepting. Neither of them names a peer. The
// client names the accepted one and carries no path of its own, so the two ends
// of a connection are told apart by which one does the naming.
func TestLsofUnixSocketParser_clientAndServer(t *testing.T) {
	parser := newLsofUnixSocketParser()

	lines := []string{
		"p78879\x00",
		"f3\x00d0xe396ab59862314e8\x00n/tmp/probe.sock\x00",
		"f4\x00d0xd82e0ac85b8e6a97\x00n/tmp/probe.sock\x00",
		"p78881\x00",
		"f3\x00d0x628a5982efaac095\x00n->0xd82e0ac85b8e6a97\x00",
	}
	for _, line := range lines {
		assert.Equal(t, parser.parseLine(line), nil)
	}

	assert.Equal(t, len(parser.unixSocketsByPid), 2)
	assert.SlicesEqual(t, parser.unixSocketsByPid[78879], []UnixSocket{
		{Fd: "3", Device: "0xe396ab59862314e8", Path: "/tmp/probe.sock"},
		{Fd: "4", Device: "0xd82e0ac85b8e6a97", Path: "/tmp/probe.sock"},
	})
	assert.SlicesEqual(t, parser.unixSocketsByPid[78881], []UnixSocket{
		{Fd: "3", Device: "0x628a5982efaac095", PeerDevice: "0xd82e0ac85b8e6a97"},
	})
}

// A socketpair(2), whose two ends name one another and have no path at all: they
// came into being connected, so neither of them dialed the other.
func TestLsofUnixSocketParser_socketPair(t *testing.T) {
	parser := newLsofUnixSocketParser()

	lines := []string{
		"p1234\x00",
		"f5\x00d0x41437afcb244b221\x00n->0x28cfa77e3695bc2\x00",
		"p5678\x00",
		"f6\x00d0x28cfa77e3695bc2\x00n->0x41437afcb244b221\x00",
	}
	for _, line := range lines {
		assert.Equal(t, parser.parseLine(line), nil)
	}

	assert.SlicesEqual(t, parser.unixSocketsByPid[1234], []UnixSocket{
		{Fd: "5", Device: "0x41437afcb244b221", PeerDevice: "0x28cfa77e3695bc2"},
	})
	assert.SlicesEqual(t, parser.unixSocketsByPid[5678], []UnixSocket{
		{Fd: "6", Device: "0x28cfa77e3695bc2", PeerDevice: "0x41437afcb244b221"},
	})
}

// lsof spells a socket whose peer is gone "->(none)", which names no peer and
// must not be mistaken for one.
func TestLsofUnixSocketParser_peerGone(t *testing.T) {
	parser := newLsofUnixSocketParser()

	assert.Equal(t, parser.parseLine("p1234\x00"), nil)
	assert.Equal(t, parser.parseLine("f0\x00d0x520254f296b152e3\x00n->(none)\x00"), nil)

	assert.SlicesEqual(t, parser.unixSocketsByPid[1234], []UnixSocket{
		{Fd: "0", Device: "0x520254f296b152e3"},
	})
}

// Not every path lsof reports is one: a socket bound to a relative path is
// named by that path, with no leading slash to recognize it by. There is one on
// the laptop this was measured on, out of 82 paths.
//
// So a name is a peer when it starts with "->" and a path in every other case.
// Testing for a leading slash instead would drop this one on the floor.
func TestLsofUnixSocketParser_relativePath(t *testing.T) {
	parser := newLsofUnixSocketParser()

	assert.Equal(t, parser.parseLine("p1234\x00"), nil)
	assert.Equal(t, parser.parseLine("f34\x00d0x1c58c388a7ab6898\x00ndocker-desktop-build.sock\x00"), nil)

	assert.SlicesEqual(t, parser.unixSocketsByPid[1234], []UnixSocket{
		{Fd: "34", Device: "0x1c58c388a7ab6898", Path: "docker-desktop-build.sock"},
	})
}

// A socket path can contain spaces, five of them doing so on the laptop this was
// measured on. The fields are NUL terminated, so a space is nothing special —
// but only as long as nothing starts splitting them on whitespace.
func TestLsofUnixSocketParser_pathWithSpaces(t *testing.T) {
	parser := newLsofUnixSocketParser()

	path := "/Users/johan/Library/Application Support/Code/1.13-main.sock"

	assert.Equal(t, parser.parseLine("p1234\x00"), nil)
	assert.Equal(t, parser.parseLine("f3\x00d0x1111\x00n"+path+"\x00"), nil)

	assert.SlicesEqual(t, parser.unixSocketsByPid[1234], []UnixSocket{
		{Fd: "3", Device: "0x1111", Path: path},
	})
}

// Every socket belongs to the process lsof last named, so a socket arriving
// before any process at all is output we don't understand.
func TestLsofUnixSocketParser_socketBeforeAnyPid(t *testing.T) {
	parser := newLsofUnixSocketParser()

	err := parser.parseLine("f3\x00d0x628a5982efaac095\x00n->0xd82e0ac85b8e6a97\x00")

	assert.Equal(t, err != nil, true)
}

func TestLsofUnixSocketParser_unparseablePid(t *testing.T) {
	parser := newLsofUnixSocketParser()

	err := parser.parseLine("pgurka\x00")

	assert.Equal(t, err != nil, true)
}

// The real lsof should report a real connected pair the way the parser tests
// above say it does: the server's accepted socket named by its path, and the
// client naming that socket's device.
//
// Both ends are held by this very process, which is exactly what a self
// connection is and costs the test nothing: the listing does not care which
// process holds which end.
//
// macOS only, unlike the real-lsof pipe tests, and not for lack of trying to
// write it platform independently. The peer edge is the whole of what this
// asserts, and on Linux there is none to assert: lsof reports a unix socket's
// own kernel address and stops there, that field being /proc/net/unix's Num
// column reformatted, and neither source knows a peer. Supplying it from a
// netlink sock_diag dump is a slice of its own.
func TestGetUnixSocketsByPid(t *testing.T) {
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

	ourSockets := unixSocketsByPid[os.Getpid()]

	// The listener and the socket accepted on it, both named by the path
	ourPath := map[string]bool{}
	for _, socket := range ourSockets {
		if socket.Path != path {
			continue
		}

		ourPath[socket.Device] = true
	}

	// The client's socket, which carries no path and names one of those two
	namesOurPath := false
	for _, socket := range ourSockets {
		if !ourPath[socket.PeerDevice] {
			continue
		}

		assert.Equal(t, socket.Path, "")

		namesOurPath = true
	}

	if len(ourPath) == 0 || !namesOurPath {
		t.Fatalf("lsof reported %v, expected a socket named %s and a socket naming it",
			ourSockets, path)
	}
}
