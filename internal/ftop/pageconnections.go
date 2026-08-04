package ftop

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/walles/ftop/internal/processes"
)

// Writes one line per connection, with the arrows pointing from whoever dialed
// to whoever was dialed. That is the one directional fact about a connection
// worth knowing, and it tells the reader which side is the service. Connections
// nobody can tell the direction of, every UDP one included, get an arrow pointing
// both ways instead.
//
// peerLabel is what to call the peer of a connection. It is never asked about a
// listening port, since nobody is at the other end of one of those.
//
// The connections are written in the order they come in, so hand them over
// sorted, see processes.NetworkConnections().
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
		dialer string

		// Us, plus whoever we dialed
		middle      string
		fancyMiddle string

		description string
	}

	us := currentProcess.String()
	fancyUs := u.highlight(us)

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
			line.dialer = peerLabel(connection.Peer)

		default:
			// Both arrows are the same number of columns wide, so which one a line
			// gets doesn't disturb the alignment of the lines around it.
			arrow := " --> "
			if connection.Direction == processes.DirectionUnknown {
				arrow = " <-> "
			}

			peer := peerLabel(connection.Peer)
			line.middle = us + arrow + peer
			line.fancyMiddle = fancyUs + arrow + peer
		}

		dialerWidth = max(dialerWidth, utf8.RuneCountInString(line.dialer))
		middleWidth = max(middleWidth, utf8.RuneCountInString(line.middle))

		lines = append(lines, line)
	}

	for _, line := range lines {
		dialer := ""
		if dialerWidth > 0 {
			arrow := " --> "
			if line.dialer == "" {
				arrow = strings.Repeat(" ", utf8.RuneCountInString(arrow))
			}

			dialer = rightPadded(line.dialer, dialerWidth) + arrow
		}

		middle := line.fancyMiddle + strings.Repeat(" ", middleWidth-utf8.RuneCountInString(line.middle))

		// Two spaces between columns, matching launchHierarchyForPaging()
		pt.writeLine(dialer + middle + "  " + line.description)
	}
}

// The rightmost column of a connection line: the protocol and the port being
// served, plus whatever makes this line more than one plain connection.
//
// Never an address: for a connection between processes the address is always
// loopback and says nothing, and for a remote peer it is in the peer column
// already.
func connectionDescription(connection processes.Connection) string {
	description := string(connection.Protocol) + " " + strconv.Itoa(connection.Port)

	if connection.Listening {
		return description + " (listening)"
	}

	if connection.Count > 1 {
		return description + " (×" + strconv.Itoa(connection.Count) + ")"
	}

	return description
}

// s with spaces appended until it is width columns wide, or s itself if it is
// that wide already.
func rightPadded(s string, width int) string {
	padding := max(0, width-utf8.RuneCountInString(s))

	return s + strings.Repeat(" ", padding)
}
