package ftop

import (
	"errors"
	"strings"
	"testing"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

// No pipes, for the tests that are about sockets
var noPipes = pipeListing{byPid: map[int][]processes.PipeEnd{}}

// No sockets, for the tests that are about pipes
var noSockets = socketListing{byPid: map[int][]processes.Socket{}}

// Connections to other processes, one line each, with the arrows pointing from
// whoever dialed to whoever was dialed — both ways for the UDP peer, since UDP says
// nothing about who dialed whom. The listening socket on 8080 and the connection to
// 1.2.3.4 belong in the Network Connections section and must not turn up here.
func TestIpcConnectionsForPagingListsProcessPeers(t *testing.T) {
	sockets := socketListing{byPid: map[int][]processes.Socket{
		42: {
			{Fd: "3", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:8080", Listening: true},
			{Fd: "4", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"},
			{Fd: "5", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:54322", Remote: "127.0.0.1:22"},
			{Fd: "6", Protocol: processes.ProtocolTcp, Local: "192.168.50.32:50000", Remote: "1.2.3.4:443"},
			{Fd: "7", Protocol: processes.ProtocolUdp, Local: "127.0.0.1:51293", Remote: "127.0.0.1:53"},
		},
		999: {{Fd: "7", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"}},
		1: {
			{Fd: "9", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:22", Listening: true},
			{Fd: "10", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:22", Remote: "127.0.0.1:54322"},
		},
		777: {{Fd: "3", Protocol: processes.ProtocolUdp, Local: "127.0.0.1:53", Remote: "127.0.0.1:51293"}},
	}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{
		picked,
		{Pid: 999, Cmdline: "curl"},
		{Pid: 1, Cmdline: "sshd"},
		{Pid: 777, Cmdline: "dnsmasq"},
	}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, noPipes, &pt)

	expected := "" +
		"<Detected: TCP, UDP, pipes. Not detected: unix sockets>\n" +
		"curl(999) --> picked(42)                   tcp 8080\n" +
		"              picked(42) --> sshd(1)       tcp 22\n" +
		"              picked(42) <-> dnsmasq(777)  udp 53\n"
	assert.Equal(t, sectionBody(page.String()), expected)

	assert.Equal(t, stringsContains(page.String(), "──Inter Process Communication──"), true)
}

// Pipes and sockets share the section, and the lines are sorted together rather
// than one kind after the other: incoming first, then outgoing, and each of
// those blocks stays one protocol at a time.
//
// Data flows from the writer to the reader, so the writing end is the outgoing
// one — which makes the arrow point the way the data goes.
func TestIpcConnectionsForPagingListsPipes(t *testing.T) {
	sockets := socketListing{byPid: map[int][]processes.Socket{
		42: {{Fd: "5", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:54322", Remote: "127.0.0.1:22"}},
		1: {
			{Fd: "9", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:22", Listening: true},
			{Fd: "10", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:22", Remote: "127.0.0.1:54322"},
		},
	}}

	pipes := pipeListing{byPid: map[int][]processes.PipeEnd{
		1234: {{Fd: "1", Access: processes.PipeAccessWrite, Inode: "16466"}},
		42: {
			{Fd: "0", Access: processes.PipeAccessRead, Inode: "16466"},
			{Fd: "1", Access: processes.PipeAccessWrite, Inode: "16467"},
		},
		5678: {{Fd: "0", Access: processes.PipeAccessRead, Inode: "16467"}},
	}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{
		picked,
		{Pid: 1, Cmdline: "sshd"},
		{Pid: 1234, Cmdline: "grep"},
		{Pid: 5678, Cmdline: "sort"},
	}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, pipes, &pt)

	expected := "" +
		"<Detected: TCP, UDP, pipes. Not detected: unix sockets>\n" +
		"grep(1234) --> picked(42)                 pipe\n" +
		"               picked(42) --> sort(5678)  pipe\n" +
		"               picked(42) --> sshd(1)     tcp 22\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// macOS lsof won't say which end of a pipe writes, so there the arrow points
// both ways, the way it does for UDP.
func TestIpcConnectionsForPagingPipeOfUnknownDirection(t *testing.T) {
	pipes := pipeListing{byPid: map[int][]processes.PipeEnd{
		42:   {{Fd: "1", Device: "0xaaaa", PeerDevice: "0xbbbb"}},
		5678: {{Fd: "0", Device: "0xbbbb", PeerDevice: "0xaaaa"}},
	}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{picked, {Pid: 5678, Cmdline: "sort"}}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, allProcesses, noSockets, pipes, &pt)

	expected := "" +
		"<Detected: TCP, UDP, pipes. Not detected: unix sockets>\n" +
		"picked(42) <-> sort(5678)  pipe\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// A pipe has no port, so its description is the bare word "pipe". Several pipes
// to the same process still aggregate into one line with a count.
func TestIpcConnectionsForPagingSeveralPipesToOnePeer(t *testing.T) {
	pipes := pipeListing{byPid: map[int][]processes.PipeEnd{
		42: {
			{Fd: "1", Access: processes.PipeAccessWrite, Inode: "16466"},
			{Fd: "2", Access: processes.PipeAccessWrite, Inode: "16467"},
		},
		5678: {
			{Fd: "0", Access: processes.PipeAccessRead, Inode: "16466"},
			{Fd: "3", Access: processes.PipeAccessRead, Inode: "16467"},
		},
	}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{picked, {Pid: 5678, Cmdline: "sort"}}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, allProcesses, noSockets, pipes, &pt)

	expected := "" +
		"<Detected: TCP, UDP, pipes. Not detected: unix sockets>\n" +
		"picked(42) --> sort(5678)  pipe (×2)\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// The picked process is what the whole page is about, and every other section
// points it out the same way.
func TestIpcConnectionsForPagingHighlightsThePickedProcess(t *testing.T) {
	sockets := socketListing{byPid: map[int][]processes.Socket{
		42:  {{Fd: "3", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"}},
		999: {{Fd: "7", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"}},
	}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{picked, {Pid: 999, Cmdline: "curl"}}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, noPipes, &pt)

	assert.Equal(t, stringsContains(page.String(), ui.highlight("picked(42)")), true)
}

// lsof runs after the process listing, so a peer can be a process we have no
// name for. Its PID is still worth showing.
func TestIpcConnectionsForPagingNamelessPeer(t *testing.T) {
	sockets := socketListing{byPid: map[int][]processes.Socket{
		42: {
			{Fd: "3", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:8080", Listening: true},
			{Fd: "4", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"},
		},
		999: {{Fd: "7", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"}},
	}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, []*processes.Process{picked}, sockets, noPipes, &pt)

	expected := "" +
		"<Detected: TCP, UDP, pipes. Not detected: unix sockets>\n" +
		"PID 999 --> picked(42)  tcp 8080\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// With nobody dialing us there is no left hand column, and the lines start where
// the process does rather than indented past an arrow nothing needs.
func TestIpcConnectionsForPagingOutgoingOnly(t *testing.T) {
	sockets := socketListing{byPid: map[int][]processes.Socket{
		42: {{Fd: "3", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:54322", Remote: "127.0.0.1:22"}},
		1:  {{Fd: "9", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:22", Remote: "127.0.0.1:54322"}},
	}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{picked, {Pid: 1, Cmdline: "sshd"}}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, noPipes, &pt)

	expected := "" +
		"<Detected: TCP, UDP, pipes. Not detected: unix sockets>\n" +
		"picked(42) --> sshd(1)  tcp 22\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// The caveat has to be above the connections: it changes how they are read, and
// this page goes into a pager where a reader may never reach the bottom.
func TestIpcConnectionsForPagingNoConnections(t *testing.T) {
	picked := &processes.Process{Pid: 42, Cmdline: "picked"}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, []*processes.Process{picked}, noSockets, noPipes, &pt)

	expected := "" +
		"<Detected: TCP, UDP, pipes. Not detected: unix sockets>\n" +
		"<No connections found>\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// With no listing at all there is nothing for the caveat to be a caveat about,
// so the errors are all there is to say.
func TestIpcConnectionsForPagingShowsErrors(t *testing.T) {
	sockets := socketListing{err: errors.New("boom")}
	pipes := pipeListing{err: errors.New("bang")}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, []*processes.Process{picked}, sockets, pipes, &pt)

	expected := "" +
		"<Unable to list sockets: boom>\n" +
		"<Unable to list pipes: bang>\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// Two lsof invocations mean either one can fail on its own. What did come back
// is still worth showing, and the caveat says what this listing is missing on
// top of what isn't implemented at all.
func TestIpcConnectionsForPagingPipeListingFailed(t *testing.T) {
	sockets := socketListing{byPid: map[int][]processes.Socket{
		42: {{Fd: "3", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:54322", Remote: "127.0.0.1:22"}},
		1:  {{Fd: "9", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:22", Remote: "127.0.0.1:54322"}},
	}}
	pipes := pipeListing{err: errors.New("boom")}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{picked, {Pid: 1, Cmdline: "sshd"}}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, pipes, &pt)

	expected := "" +
		"<Unable to list pipes: boom>\n" +
		"<Detected: TCP, UDP. Not detected: unix sockets>\n" +
		"picked(42) --> sshd(1)  tcp 22\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// One listing failed and the other found nothing. Both facts matter, and
// neither one is a reason to stop before saying the other.
func TestIpcConnectionsForPagingPipeListingFailedAndNoSockets(t *testing.T) {
	pipes := pipeListing{err: errors.New("boom")}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, []*processes.Process{picked}, noSockets, pipes, &pt)

	expected := "" +
		"<Unable to list pipes: boom>\n" +
		"<Detected: TCP, UDP. Not detected: unix sockets>\n" +
		"<No connections found>\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

func TestIpcConnectionsForPagingSocketListingFailed(t *testing.T) {
	sockets := socketListing{err: errors.New("boom")}
	pipes := pipeListing{byPid: map[int][]processes.PipeEnd{
		42:   {{Fd: "1", Access: processes.PipeAccessWrite, Inode: "16466"}},
		5678: {{Fd: "0", Access: processes.PipeAccessRead, Inode: "16466"}},
	}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{picked, {Pid: 5678, Cmdline: "sort"}}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, pipes, &pt)

	expected := "" +
		"<Unable to list sockets: boom>\n" +
		"<Detected: pipes. Not detected: unix sockets>\n" +
		"picked(42) --> sort(5678)  pipe\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}
