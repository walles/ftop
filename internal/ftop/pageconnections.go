package ftop

import (
	"strconv"
	"strings"

	"github.com/rivo/uniseg"
	"github.com/walles/ftop/internal/processes"
)

// Writes one line per connection, with the arrows pointing the way
// processes.Direction means them: from whoever dialed to whoever was dialed for a
// socket, which tells the reader which side is the service, and from the writer
// to the reader for a pipe, which is the way the data goes. Connections nobody
// can tell the direction of, every UDP one included, get "◀?▶" instead of an
// arrow.
//
// peerLabel is what to call the peer of a connection. It is never asked about a
// listening port, since nobody is at the other end of one of those.
//
// currentProcess is highlighted wherever it turns up, which includes the peer
// column when it is connected to itself. A peer is that process when it carries
// its PID, so a caller whose peers are not local processes, and which therefore
// leaves Peer.Pid at 0, never gets a highlighted peer.
//
// The connections are written in the order they come in, so hand them over
// sorted, see processes.SortConnections().
//
// The columns are only as wide as these connections need them to be, and are
// measured over these connections alone: two sections' worth of lines are far
// enough apart on the page for a few columns of offset between them to be
// invisible. With nobody dialing in there is no left hand column at all, and the
// lines start at currentProcess rather than indented past an arrow nothing uses.
func (u *Ui) writeConnectionLines(
	currentProcess *processes.Process,
	connections []processes.Connection,
	peerLabel func(peer processes.Peer) string,
	pt *pageText,
) {
	// One line each, in plain text for measuring and styled for writing, per the
	// launchHierarchyForPaging() idiom
	type connectionLine struct {
		// Whoever dialed us, empty unless somebody did
		dialer      string
		fancyDialer string

		// Us, plus whoever we dialed
		middle      string
		fancyMiddle string

		description string
	}

	us := currentProcess.String()
	fancyUs := u.highlight(us)

	// A peer PID seen on more than one line gets called out, so that e.g. two
	// same-named processes with different PIDs don't read as one.
	peerPidCounts := make(map[int]int, len(connections))
	for _, connection := range connections {
		if connection.Listening || connection.Peer.Pid == 0 || connection.Peer.Pid == currentProcess.Pid {
			continue
		}

		peerPidCounts[connection.Peer.Pid]++
	}

	// By PID rather than by label: a peer is the same process or it isn't,
	// whatever either of them decides to call it. Pid 0 is no process at all.
	styledPeerLabel := func(peer processes.Peer) (string, string) {
		label := peerLabel(peer)
		if peer.Pid != 0 && peer.Pid == currentProcess.Pid {
			return label, u.highlight(label)
		}

		if peerPidCounts[peer.Pid] > 1 {
			return label, u.highlightDuplicate(label)
		}

		return label, label
	}

	lines := make([]connectionLine, 0, len(connections))
	dialerWidth := 0
	middleWidth := 0

	for _, connection := range connections {
		line := connectionLine{
			middle:      us,
			fancyMiddle: fancyUs,
			description: connectionDescription(connection),
		}

		switch {
		case connection.Listening:
			// Nobody has dialed in yet, and nobody has been dialed

		case connection.Direction == processes.DirectionIncoming:
			line.dialer, line.fancyDialer = styledPeerLabel(connection.Peer)

		default:
			// All three markers are the same number of columns wide, so which one
			// a line gets doesn't disturb the alignment of the lines around it.
			arrow := " ──▶ "
			if connection.Direction == processes.DirectionUnknown {
				arrow = " ◀?▶ "
			}

			peer, fancyPeer := styledPeerLabel(connection.Peer)
			line.middle = us + arrow + peer
			line.fancyMiddle = fancyUs + arrow + fancyPeer
		}

		dialerWidth = max(dialerWidth, uniseg.StringWidth(line.dialer))
		middleWidth = max(middleWidth, uniseg.StringWidth(line.middle))

		lines = append(lines, line)
	}

	for _, line := range lines {
		dialer := ""
		if dialerWidth > 0 {
			arrow := " ──▶ "
			if line.dialer == "" {
				arrow = strings.Repeat(" ", uniseg.StringWidth(arrow))
			}

			dialer = rightPadded(line.dialer, line.fancyDialer, dialerWidth) + arrow
		}

		middle := rightPadded(line.middle, line.fancyMiddle, middleWidth)

		// Two spaces between columns, matching launchHierarchyForPaging()
		pt.writeLine(dialer + middle + "  " + line.description)
	}
}

// The rightmost column of a connection line: the protocol and the port being
// served, plus whatever makes this line more than one plain connection.
//
// A connection carried by something that has no ports, a pipe being one, gets
// the bare protocol instead. A unix domain socket has a path where a network
// socket has a port, so that goes here too, and it is the one thing on the line
// naming which service of a process' several a connection reaches.
//
// Never an address: for a connection between processes the address is always
// loopback and says nothing, and for a remote peer it is in the peer column
// already. A unix socket path is not an address in that sense — it is local by
// definition, and it identifies the service rather than the machine.
func connectionDescription(connection processes.Connection) string {
	description := string(connection.Protocol)
	if connection.Port != 0 {
		description += " " + strconv.Itoa(connection.Port)
	}

	if connection.Path != "" {
		description += " " + connection.Path
	}

	if connection.Listening {
		return description + " (listening)"
	}

	if connection.Count > 1 {
		return description + " (×" + strconv.Itoa(connection.Count) + ")"
	}

	return description
}
