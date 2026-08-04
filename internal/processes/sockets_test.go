package processes

import (
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/walles/ftop/internal/assert"
)

func TestLsofSocketParser(t *testing.T) {
	parser := newLsofSocketParser()

	lines := []string{
		"p7619\x00",
		"f31\x00n192.168.50.32:57759->172.217.19.234:443\x00TST=ESTABLISHED\x00TQR=0\x00TQS=0\x00",
		"p28727\x00",
		"f10\x00n*:7000\x00TST=LISTEN\x00TQR=0\x00TQS=0\x00",
	}
	for _, line := range lines {
		assert.Equal(t, parser.parseLine(line), nil)
	}

	assert.Equal(t, len(parser.socketsByPid), 2)
	assert.SlicesEqual(t, parser.socketsByPid[7619], []Socket{
		{Fd: "31", Local: "192.168.50.32:57759", Remote: "172.217.19.234:443"},
	})
	assert.SlicesEqual(t, parser.socketsByPid[28727], []Socket{
		{Fd: "10", Local: "*:7000", Listening: true},
	})
}

// Asking lsof for the TCP state gets us the send and receive queue sizes as
// well, in fields that all start with the same "T" as the state does. Telling
// them apart is what the "ST=" prefix is for.
func TestLsofSocketParser_queueSizeFields(t *testing.T) {
	parser := newLsofSocketParser()

	assert.Equal(t, parser.parseLine("p1\x00"), nil)
	assert.Equal(t, parser.parseLine("f3\x00n*:7000\x00TST=LISTEN\x00TQR=17\x00TQS=42\x00"), nil)

	assert.SlicesEqual(t, parser.socketsByPid[1], []Socket{
		{Fd: "3", Local: "*:7000", Listening: true},
	})
}

// One process can hold any number of sockets, and they arrive one line each
// after the line naming the process.
func TestLsofSocketParser_severalSocketsPerProcess(t *testing.T) {
	parser := newLsofSocketParser()

	lines := []string{
		"p28794\x00",
		"f4\x00n[fe80:4::1122]:63477->[fe80:4::4455]:49157\x00TST=ESTABLISHED\x00TQR=0\x00TQS=0\x00",
		"f6\x00n[fe80:4::1122]:62126->[fe80:4::4455]:49157\x00TST=ESTABLISHED\x00TQR=0\x00TQS=0\x00",
	}
	for _, line := range lines {
		assert.Equal(t, parser.parseLine(line), nil)
	}

	assert.SlicesEqual(t, parser.socketsByPid[28794], []Socket{
		{Fd: "4", Local: "[fe80:4::1122]:63477", Remote: "[fe80:4::4455]:49157"},
		{Fd: "6", Local: "[fe80:4::1122]:62126", Remote: "[fe80:4::4455]:49157"},
	})
}

// lsof can report files it has no name for. Those tell us nothing, and must not
// cost us the sockets around them.
func TestLsofSocketParser_ignoresNamelessSockets(t *testing.T) {
	parser := newLsofSocketParser()

	lines := []string{
		"p1\x00",
		"f3\x00n\x00TST=ESTABLISHED\x00TQR=0\x00TQS=0\x00",
		"f4\x00n*:7000\x00TST=LISTEN\x00TQR=0\x00TQS=0\x00",
	}
	for _, line := range lines {
		assert.Equal(t, parser.parseLine(line), nil)
	}

	assert.SlicesEqual(t, parser.socketsByPid[1], []Socket{
		{Fd: "4", Local: "*:7000", Listening: true},
	})
}

// A socket that is neither listening nor connected has an address and nothing
// else to say. Reporting it as it is keeps the decision about what to do with
// it in one place, see NetworkConnections().
func TestLsofSocketParser_unconnectedSocket(t *testing.T) {
	parser := newLsofSocketParser()

	assert.Equal(t, parser.parseLine("p1\x00"), nil)
	assert.Equal(t, parser.parseLine("f3\x00n*:60000\x00TST=CLOSED\x00TQR=0\x00TQS=0\x00"), nil)

	assert.SlicesEqual(t, parser.socketsByPid[1], []Socket{
		{Fd: "3", Local: "*:60000"},
	})
}

// Sockets belong to the process lsof most recently named, so a socket arriving
// before any process at all is something we can't make sense of.
func TestLsofSocketParser_socketBeforeAnyPid(t *testing.T) {
	parser := newLsofSocketParser()

	err := parser.parseLine("f3\x00n*:7000\x00TST=LISTEN\x00")

	assert.Equal(t, err == nil, false)
}

func TestLsofSocketParser_unparseablePid(t *testing.T) {
	parser := newLsofSocketParser()

	err := parser.parseLine("pnotanumber\x00")

	assert.Equal(t, err == nil, false)
}

// The real lsof should be able to see a socket we just opened ourselves.
func TestGetSocketsByPid(t *testing.T) {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof not available: ", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = listener.Close()
	}()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	socketsByPid, err := GetSocketsByPid()
	if err != nil {
		t.Fatalf("listing sockets failed: %v", err)
	}

	foundOurListener := false
	for _, socket := range socketsByPid[os.Getpid()] {
		if !socket.Listening {
			continue
		}

		if !strings.HasSuffix(socket.Local, ":"+port) {
			continue
		}

		foundOurListener = true
	}

	assert.Equal(t, foundOurListener, true)
}
