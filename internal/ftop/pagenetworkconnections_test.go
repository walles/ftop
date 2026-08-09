package ftop

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

// Replaces the reverse DNS lookups for the duration of the test. The returned
// function tells which addresses we were asked to resolve.
func fakeDns(t *testing.T, names map[string]string) func() []string {
	t.Helper()

	original := resolveAddresses
	t.Cleanup(func() {
		resolveAddresses = original
	})

	var asked []string
	resolveAddresses = func(addresses []string) map[string]string {
		asked = append(asked, addresses...)
		return names
	}

	return func() []string {
		return asked
	}
}

// Listening ports first, then who dialed in, then who we dialed. The 12 incoming
// connections all come from the same address and collapse into one counted line,
// 1.2.3.4 resolving to nothing renders as the address it already is, and the
// connection to sshd belongs in the Inter Process Communication section.
func TestNetworkConnectionsForPagingListsRemotePeers(t *testing.T) {
	mySockets := []processes.Socket{
		{Fd: "100", Protocol: processes.ProtocolTcp, Local: "192.168.50.32:8080", Listening: true},
		{Fd: "400", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:60000", Remote: "127.0.0.1:22"},
	}
	for i := range 12 {
		mySockets = append(mySockets, processes.Socket{
			Fd:       fmt.Sprintf("2%02d", i),
			Protocol: processes.ProtocolTcp,
			Local:    "192.168.50.32:8080",
			Remote:   fmt.Sprintf("1.2.3.4:%d", 1000+i),
		})
	}
	for i := range 7 {
		mySockets = append(mySockets, processes.Socket{
			Fd:       fmt.Sprintf("3%02d", i),
			Protocol: processes.ProtocolTcp,
			Local:    fmt.Sprintf("192.168.50.32:%d", 50000+i),
			Remote:   "140.82.114.25:443",
		})
	}
	sockets := socketListing{byPid: map[int][]processes.Socket{
		42: mySockets,
		1:  {{Fd: "9", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:22", Remote: "127.0.0.1:60000"}},
	}}
	fakeDns(t, map[string]string{"140.82.114.25": "api.github.com"})

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{picked, {Pid: 1, Cmdline: "sshd"}}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.networkConnectionsForPaging(picked, allProcesses, sockets, &pt)

	expected := "" +
		"            picked(42)                     tcp 8080 (listening)\n" +
		"1.2.3.4 ──▶ picked(42)                     tcp 8080 (×12)\n" +
		"            picked(42) ──▶ api.github.com  tcp 443 (×7)\n"
	assert.Equal(t, sectionBody(page.String()), expected)

	assert.Equal(t, stringsContains(page.String(), "──Network Connections──"), true)
}

// UDP says nothing about who dialed whom, so its lines get a question mark rather
// than an arrow. That marker is as wide as the arrow, so a section holding both
// kinds of line still lines up.
func TestNetworkConnectionsForPagingUndeterminedDirection(t *testing.T) {
	sockets := socketListing{byPid: map[int][]processes.Socket{
		42: {
			{Fd: "3", Protocol: processes.ProtocolTcp, Local: "192.168.50.32:50000", Remote: "140.82.114.25:443"},
			{Fd: "4", Protocol: processes.ProtocolUdp, Local: "192.168.50.32:51293", Remote: "8.8.8.8:53"},
		},
	}}
	fakeDns(t, map[string]string{"140.82.114.25": "api.github.com", "8.8.8.8": "dns.google"})

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.networkConnectionsForPaging(picked, []*processes.Process{picked}, sockets, &pt)

	expected := "" +
		"picked(42) ──▶ api.github.com  tcp 443\n" +
		"picked(42) ◀?▶ dns.google      udp 53\n"
	assert.Equal(t, sectionBody(page.String()), expected)
}

// Only remote hosts have addresses to resolve. Asking about anything else would
// mean paying the reverse DNS timeout for nothing.
func TestNetworkConnectionsForPagingResolvesRemotePeersOnly(t *testing.T) {
	sockets := socketListing{byPid: map[int][]processes.Socket{
		42: {
			{Fd: "3", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:8080", Listening: true},
			{Fd: "4", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:8080", Remote: "127.0.0.1:54321"},
			{Fd: "5", Protocol: processes.ProtocolTcp, Local: "192.168.50.32:50000", Remote: "1.2.3.4:443"},
		},
		999: {{Fd: "7", Protocol: processes.ProtocolTcp, Local: "127.0.0.1:54321", Remote: "127.0.0.1:8080"}},
	}}
	asked := fakeDns(t, nil)

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}
	allProcesses := []*processes.Process{picked, {Pid: 999, Cmdline: "curl"}}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.networkConnectionsForPaging(picked, allProcesses, sockets, &pt)

	assert.SlicesEqual(t, asked(), []string{"1.2.3.4"})
}

// A process talking to nobody gets a line saying so, rather than a section that
// looks unfinished.
func TestNetworkConnectionsForPagingNoConnections(t *testing.T) {
	sockets := socketListing{byPid: map[int][]processes.Socket{}}
	fakeDns(t, nil)

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.networkConnectionsForPaging(picked, []*processes.Process{picked}, sockets, &pt)

	assert.Equal(t, sectionBody(page.String()), "<No connections found>\n")
}

func TestNetworkConnectionsForPagingShowsErrors(t *testing.T) {
	sockets := socketListing{err: errors.New("boom")}
	fakeDns(t, nil)

	picked := &processes.Process{Pid: 42, Cmdline: "picked"}

	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "")
	var page strings.Builder
	pt := pageText{out: &page}

	ui.networkConnectionsForPaging(picked, []*processes.Process{picked}, sockets, &pt)

	assert.Equal(t, sectionBody(page.String()), "<Unable to list sockets: boom>\n")
}
