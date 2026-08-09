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

// No unix sockets, for the tests that are about the other two kinds
var noUnixSockets = unixSocketListing{byPid: map[int][]processes.UnixSocket{}}

// Connections to other processes, one line each, with the arrows pointing from
// whoever dialed to whoever was dialed — a question mark for the UDP peer, since
// UDP says nothing about who dialed whom. The listening socket on 8080 and the
// connection to 1.2.3.4 belong in the Network Connections section and must not
// turn up here.
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

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, noPipes, noUnixSockets, &pt)

	expected := "" +
		"curl(999) ──▶ picked(42)                   tcp 8080\n" +
		"              picked(42) ──▶ sshd(1)       tcp 22\n" +
		"              picked(42) ◀?▶ dnsmasq(777)  udp 53\n"
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

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, pipes, noUnixSockets, &pt)

	expected := "" +
		"grep(1234) ──▶ picked(42)                 pipe\n" +
		"               picked(42) ──▶ sort(5678)  pipe\n" +
		"               picked(42) ──▶ sshd(1)     tcp 22\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// A pipe we could establish no access mode for gets a question mark instead of
// an arrow, the way a UDP connection does. Both ends here are shaped the way
// macOS lsof reports an anonymous pipe, which is the listing that carries no
// access modes of its own.
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

	ui.ipcConnectionsForPaging(picked, allProcesses, noSockets, pipes, noUnixSockets, &pt)

	expected := "picked(42) ◀?▶ sort(5678)  pipe\n"
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

	ui.ipcConnectionsForPaging(picked, allProcesses, noSockets, pipes, noUnixSockets, &pt)

	expected := "picked(42) ──▶ sort(5678)  pipe (×2)\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// A unix domain socket connection is named by the path it was made over, which
// goes where a network socket's port goes: it is what says which of a server's
// several services this connection reaches.
//
// The client is the end that named the other, so the arrow points at the server.
func TestIpcConnectionsForPagingListsUnixSockets(t *testing.T) {
	unixSockets := unixSocketListing{byPid: map[int][]processes.UnixSocket{
		42: {{Fd: "3", Device: "0x3333", PeerDevice: "0x2222"}},
		1: {
			{Fd: "3", Device: "0x1111", Path: "/var/run/docker.sock"},
			{Fd: "4", Device: "0x2222", Path: "/var/run/docker.sock"},
		},
	}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{picked, {Pid: 1, Cmdline: "dockerd"}}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, allProcesses, noSockets, noPipes, unixSockets, &pt)

	expected := "picked(42) ──▶ dockerd(1)  unix /var/run/docker.sock\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// The same connection from the server's side. Its accepted socket knows the
// path but not who dialed it, so the client naming it is the whole of what ties
// the two together — and the line comes out reading the same way round.
func TestIpcConnectionsForPagingUnixSocketServerSide(t *testing.T) {
	unixSockets := unixSocketListing{byPid: map[int][]processes.UnixSocket{
		999: {{Fd: "3", Device: "0x3333", PeerDevice: "0x2222"}},
		42: {
			{Fd: "3", Device: "0x1111", Path: "/var/run/docker.sock"},
			{Fd: "4", Device: "0x2222", Path: "/var/run/docker.sock"},
		},
	}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{picked, {Pid: 999, Cmdline: "curl"}}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, allProcesses, noSockets, noPipes, unixSockets, &pt)

	expected := "curl(999) ──▶ picked(42)  unix /var/run/docker.sock\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// A socketpair(2) has no path and no dialer: it was never made over the file
// system, and the two ends came into being connected. So the description is the
// bare word "unix" and the arrow is a question mark.
func TestIpcConnectionsForPagingUnixSocketPair(t *testing.T) {
	unixSockets := unixSocketListing{byPid: map[int][]processes.UnixSocket{
		42:   {{Fd: "5", Device: "0xaaaa", PeerDevice: "0xbbbb"}},
		5678: {{Fd: "0", Device: "0xbbbb", PeerDevice: "0xaaaa"}},
	}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{picked, {Pid: 5678, Cmdline: "worker"}}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, allProcesses, noSockets, noPipes, unixSockets, &pt)

	expected := "picked(42) ◀?▶ worker(5678)  unix\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// Several pipes to one peer aggregate into one line with a count, and so do
// several unix socket connections over one path.
func TestIpcConnectionsForPagingSeveralUnixSocketsToOnePeer(t *testing.T) {
	unixSockets := unixSocketListing{byPid: map[int][]processes.UnixSocket{
		42: {
			{Fd: "3", Device: "0x3333", PeerDevice: "0x2222"},
			{Fd: "4", Device: "0x4444", PeerDevice: "0x5555"},
		},
		1: {
			{Fd: "4", Device: "0x2222", Path: "/var/run/docker.sock"},
			{Fd: "5", Device: "0x5555", Path: "/var/run/docker.sock"},
		},
	}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{picked, {Pid: 1, Cmdline: "dockerd"}}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, allProcesses, noSockets, noPipes, unixSockets, &pt)

	expected := "picked(42) ──▶ dockerd(1)  unix /var/run/docker.sock (×2)\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// All four kinds in one section, sorted together rather than one kind after the
// other: whoever dialed us first, then whoever we dialed, then the ones nobody
// can tell the direction of. Each of those blocks stays one protocol at a time.
func TestIpcConnectionsForPagingListsEveryKind(t *testing.T) {
	sockets := socketListing{byPid: map[int][]processes.Socket{
		42: {
			{Fd: "3", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:8080", Listening: true},
			{Fd: "4", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"},
			{Fd: "5", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:54322", Remote: "127.0.0.1:22"},
			{Fd: "6", Protocol: processes.ProtocolUdp, Local: "127.0.0.1:51293", Remote: "127.0.0.1:53"},
		},
		999: {{Fd: "7", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"}},
		1: {
			{Fd: "9", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:22", Listening: true},
			{Fd: "10", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:22", Remote: "127.0.0.1:54322"},
		},
		777: {{Fd: "3", Protocol: processes.ProtocolUdp, Local: "127.0.0.1:53", Remote: "127.0.0.1:51293"}},
	}}

	pipes := pipeListing{byPid: map[int][]processes.PipeEnd{
		42:   {{Fd: "1", Access: processes.PipeAccessWrite, Inode: "16466"}},
		5678: {{Fd: "0", Access: processes.PipeAccessRead, Inode: "16466"}},
	}}

	unixSockets := unixSocketListing{byPid: map[int][]processes.UnixSocket{
		42: {{Fd: "3", Device: "0x3333", PeerDevice: "0x2222"}},
		1:  {{Fd: "4", Device: "0x2222", Path: "/var/run/dbus.sock"}},
	}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{
		picked,
		{Pid: 1, Cmdline: "sshd"},
		{Pid: 777, Cmdline: "dnsmasq"},
		{Pid: 999, Cmdline: "curl"},
		{Pid: 5678, Cmdline: "sort"},
	}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, pipes, unixSockets, &pt)

	expected := "" +
		"curl(999) ──▶ picked(42)                   tcp 8080\n" +
		"              picked(42) ──▶ sort(5678)    pipe\n" +
		"              picked(42) ──▶ sshd(1)       tcp 22\n" +
		"              picked(42) ──▶ sshd(1)       unix /var/run/dbus.sock\n" +
		"              picked(42) ◀?▶ dnsmasq(777)  udp 53\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// Three lsof invocations mean any one of them can fail on its own. What did come
// back is still worth showing, below an error naming what didn't.
func TestIpcConnectionsForPagingUnixSocketListingFailed(t *testing.T) {
	pipes := pipeListing{byPid: map[int][]processes.PipeEnd{
		42:   {{Fd: "1", Access: processes.PipeAccessWrite, Inode: "16466"}},
		5678: {{Fd: "0", Access: processes.PipeAccessRead, Inode: "16466"}},
	}}
	unixSockets := unixSocketListing{err: errors.New("boom")}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{picked, {Pid: 5678, Cmdline: "sort"}}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, allProcesses, noSockets, pipes, unixSockets, &pt)

	expected := "" +
		"<Unable to list unix sockets: boom>\n" +
		"picked(42) ──▶ sort(5678)  pipe\n"
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

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, noPipes, noUnixSockets, &pt)

	assert.Equal(t, stringsContains(page.String(), ui.highlight("picked(42)")), true)
}

// A process talking to itself names the picked process at both ends of the line,
// and both ends are the picked process, so both are highlighted.
func TestIpcConnectionsForPagingHighlightsBothEndsOfASelfConnection(t *testing.T) {
	pipes := pipeListing{byPid: map[int][]processes.PipeEnd{
		42: {
			{Fd: "3", Access: processes.PipeAccessWrite, Inode: "16466"},
			{Fd: "4", Access: processes.PipeAccessRead, Inode: "16466"},
		},
	}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, []*processes.Process{picked}, noSockets, pipes, noUnixSockets, &pt)

	assert.Equal(t, sectionBody(page.String()), "picked(42) ──▶ picked(42)  pipe\n")
	assert.Equal(t, strings.Count(page.String(), ui.highlight("picked(42)")), 2)
}

// A self connection sharing the section with connections to other processes:
// both of its ends are highlighted, and the columns line up the way they do on
// any other page. Highlighting is no excuse for losing the alignment, the
// styling being invisible to a reader counting columns.
func TestIpcConnectionsForPagingSelfConnectionAlignment(t *testing.T) {
	sockets := socketListing{byPid: map[int][]processes.Socket{
		42: {
			{Fd: "3", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:8080", Listening: true},
			{Fd: "4", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"},
			{Fd: "5", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"},
			{Fd: "6", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:54322"},
		},
		999: {{Fd: "7", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:54322", Remote: "127.0.0.1:8080"}},
	}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{picked, {Pid: 999, Cmdline: "elaborately-named-curl"}}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, noPipes, noUnixSockets, &pt)

	expected := "" +
		"elaborately-named-curl(999) ──▶ picked(42)                 tcp 8080\n" +
		"                                picked(42) ──▶ picked(42)  tcp 8080\n"
	assert.Equal(t, sectionBody(page.String()), expected)
	assert.Equal(t, strings.Count(page.String(), ui.highlight("picked(42)")), 3)
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

	ui.ipcConnectionsForPaging(picked, []*processes.Process{picked}, sockets, noPipes, noUnixSockets, &pt)

	expected := "PID 999 ──▶ picked(42)  tcp 8080\n"
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

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, noPipes, noUnixSockets, &pt)

	expected := "picked(42) ──▶ sshd(1)  tcp 22\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// A process talking to nobody gets a line saying so, rather than a section that
// looks unfinished.
func TestIpcConnectionsForPagingNoConnections(t *testing.T) {
	picked := &processes.Process{Pid: 42, Cmdline: "picked"}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, []*processes.Process{picked}, noSockets, noPipes, noUnixSockets, &pt)

	expected := "<No connections found>\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// Having looked nowhere, "no connections found" would be a lie, so the errors are
// all there is to say.
func TestIpcConnectionsForPagingShowsErrors(t *testing.T) {
	sockets := socketListing{err: errors.New("boom")}
	pipes := pipeListing{err: errors.New("bang")}
	unixSockets := unixSocketListing{err: errors.New("crash")}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, []*processes.Process{picked}, sockets, pipes, unixSockets, &pt)

	expected := "" +
		"<Unable to list sockets: boom>\n" +
		"<Unable to list pipes: bang>\n" +
		"<Unable to list unix sockets: crash>\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// Two lsof invocations mean either one can fail on its own. What did come back
// is still worth showing, below an error naming what didn't.
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

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, pipes, noUnixSockets, &pt)

	expected := "" +
		"<Unable to list pipes: boom>\n" +
		"picked(42) ──▶ sshd(1)  tcp 22\n"
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

	ui.ipcConnectionsForPaging(picked, []*processes.Process{picked}, noSockets, pipes, noUnixSockets, &pt)

	expected := "" +
		"<Unable to list pipes: boom>\n" +
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

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, pipes, noUnixSockets, &pt)

	expected := "" +
		"<Unable to list sockets: boom>\n" +
		"picked(42) ──▶ sort(5678)  pipe\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}
