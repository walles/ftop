package processes

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/util"
)

// One TCP socket held open by some process, as reported by lsof.
type Socket struct {
	// lsof's file descriptor number, "31" or similar. Unique within the process
	// holding it, which is what makes it usable for recognizing the same socket
	// reported to us twice.
	Fd string

	// Our own end of the socket: "127.0.0.1:8080", "*:8080" for a listener
	// bound to every interface, or "[::1]:8081" for IPv6.
	Local string

	// The other end, in the same format as Local. Empty for a socket that has no
	// other end: one that is listening, or one that is bound but was never
	// connected.
	Remote string

	Listening bool
}

// Maps PIDs to the TCP sockets held open by the corresponding processes.
//
// The listing is partial: processes we aren't allowed to inspect are missing
// from the map, so expect only a fraction of the running processes when not
// running as root. Processes without any TCP sockets are missing as well, which
// is most of them.
//
// UDP sockets are not included, and neither are pipes or unix domain sockets.
//
// This forks lsof, which takes a fraction of a second. Too slow for calling
// once per frame, fine for on-demand lookups.
func GetSocketsByPid() (map[int][]Socket, error) {
	parser := newLsofSocketParser()

	// -n: Don't resolve host names, they are slow and we do our own resolving
	// -P: Don't resolve port numbers, service names like "ipp" for 631 don't
	//   sort numerically
	// -w: Don't warn about processes we aren't allowed to inspect
	// -iTCP: List TCP sockets only, this is much faster than listing every open
	//   file of every process
	// -F pfnT0: Machine readable output with NUL terminated PID, file
	//   descriptor, name and TCP state fields
	commandline := []string{"lsof", "-n", "-P", "-w", "-iTCP", "-F", "pfnT0"}

	// Locale intentionally left alone, matching GetCwdsByPid()
	err := util.ExecInUsersLocale(commandline, parser.parseLine)
	if err == nil {
		return parser.socketsByPid, nil
	}

	// lsof exits non-zero as soon as anything at all went wrong, and failing to
	// inspect some process is business as usual. Whatever it did manage to
	// report is still good, so only give up if we got nothing.
	if len(parser.socketsByPid) == 0 {
		return nil, err
	}

	log.Infof("Listing TCP sockets partially failed, got %d processes' worth: %v",
		len(parser.socketsByPid), err)

	return parser.socketsByPid, nil
}

// Parses the output of "lsof -n -P -w -iTCP -F pfnT0", which comes in NUL
// terminated fields, one line per process and then one line per socket:
//
//	p7619\0
//	f31\0n192.168.50.32:57759->172.217.19.234:443\0TST=ESTABLISHED\0TQR=0\0TQS=0\0
type lsofSocketParser struct {
	socketsByPid map[int][]Socket

	// PID from the most recent "p" field, or -1 before the first one
	pid int
}

func newLsofSocketParser() lsofSocketParser {
	return lsofSocketParser{
		socketsByPid: map[int][]Socket{},
		pid:          -1,
	}
}

// Note that lsof escapes non-printable characters, newlines included, so one
// line of output is always one complete record.
func (parser *lsofSocketParser) parseLine(line string) error {
	// Filled in by the fields of this line, if this line describes a socket
	var socket Socket

	for field := range strings.SplitSeq(strings.TrimSuffix(line, "\x00"), "\x00") {
		err := parser.parseField(field, &socket)
		if err != nil {
			return err
		}
	}

	if socket.Local == "" {
		// A line naming a process, or a file lsof has no name for. Neither one
		// tells us about a socket.
		return nil
	}

	if parser.pid == -1 {
		return fmt.Errorf("lsof reported socket <%s> before any PID", socket.Local)
	}

	parser.socketsByPid[parser.pid] = append(parser.socketsByPid[parser.pid], socket)

	return nil
}

// Applies one field to socket, or to the parser itself for the PID field.
func (parser *lsofSocketParser) parseField(field string, socket *Socket) error {
	if field == "" {
		return nil
	}

	identifier := field[0]
	value := field[1:]

	switch identifier {
	case 'p':
		pid, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("unparseable lsof PID <%s>: %w", value, err)
		}

		parser.pid = pid

	case 'f':
		socket.Fd = value

	case 'n':
		// A listening socket is named by its own address alone, a connected one
		// by both ends of the connection
		socket.Local, socket.Remote, _ = strings.Cut(value, "->")

	case 'T':
		// Asking for the TCP state gets us the queue sizes as well, and all
		// three fields are called "T". The value prefix is what tells them
		// apart.
		state, isState := strings.CutPrefix(value, "ST=")
		if isState {
			socket.Listening = state == "LISTEN"
		}
	}

	// lsof can emit fields we didn't ask for, just ignore those
	return nil
}
