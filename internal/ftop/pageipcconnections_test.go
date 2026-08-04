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

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, &pt)

	expected := "" +
		"<Detected: TCP, UDP. Not detected: pipes, unix sockets>\n" +
		"curl(999) --> picked(42)                   tcp 8080\n" +
		"              picked(42) --> sshd(1)       tcp 22\n" +
		"              picked(42) <-> dnsmasq(777)  udp 53\n"
	assert.Equal(t, sectionBody(page.String()), expected)

	assert.Equal(t, stringsContains(page.String(), "──Inter Process Communication──"), true)
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

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, &pt)

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

	ui.ipcConnectionsForPaging(picked, []*processes.Process{picked}, sockets, &pt)

	expected := "" +
		"<Detected: TCP, UDP. Not detected: pipes, unix sockets>\n" +
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

	ui.ipcConnectionsForPaging(picked, allProcesses, sockets, &pt)

	expected := "" +
		"<Detected: TCP, UDP. Not detected: pipes, unix sockets>\n" +
		"picked(42) --> sshd(1)  tcp 22\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// The caveat has to be above the connections: it changes how they are read, and
// this page goes into a pager where a reader may never reach the bottom.
func TestIpcConnectionsForPagingNoConnections(t *testing.T) {
	sockets := socketListing{byPid: map[int][]processes.Socket{}}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, []*processes.Process{picked}, sockets, &pt)

	expected := "" +
		"<Detected: TCP, UDP. Not detected: pipes, unix sockets>\n" +
		"<No connections found>\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// Without a socket listing there is nothing for the caveat to be a caveat about,
// so the error is all there is to say.
func TestIpcConnectionsForPagingShowsErrors(t *testing.T) {
	sockets := socketListing{err: errors.New("boom")}

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.ipcConnectionsForPaging(picked, []*processes.Process{picked}, sockets, &pt)

	assert.Equal(t, sectionBody(page.String()), "<Unable to list sockets: boom>\n")
}
