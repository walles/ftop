package processes

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/util"
)

// One TCP or UDP socket held open by some process, as reported by lsof.
type Socket struct {
	// lsof's file descriptor number, "31" or similar. Not an identity: one socket
	// is reported once per descriptor it is open on, so the same socket arrives
	// under several of these. See deduplicateBySocket() for what does identify one.
	Fd string

	Protocol Protocol

	// Our own end of the socket: "127.0.0.1:8080", "*:8080" for a listener
	// bound to every interface, or "[::1]:8081" for IPv6.
	Local string

	// The other end, in the same format as Local. Empty for a socket that has no
	// other end: one that is listening, or one that is bound but was never
	// connected.
	Remote string

	// True for a socket lsof reports as listening. Never true for UDP, which has
	// no listening state.
	Listening bool
}

// Maps PIDs to the TCP and UDP sockets held open by the corresponding
// processes.
//
// The listing is partial: processes we aren't allowed to inspect are missing
// from the map, so expect only a fraction of the running processes when not
// running as root. Processes without any sockets are missing as well, which is
// most of them.
//
// Pipes and unix domain sockets are not included.
//
// This forks lsof, which takes a fraction of a second, and the filter is what
// makes it worth a fork of its own rather than a share of the unfiltered listing
// GetPipeEndsByPid() has to use — see there for the two of them timed side by
// side on macOS. It earns itself on Linux as well, by less: in a Debian container
// holding 819 socket lines out of 4122 open files, "-iTCP" took 0.047 s for 55 KB
// where full lsof took 0.085 s for 180 KB, three runs each and under 0.02 s of
// spread. A container understates it, full lsof being the side that grows with
// every file descriptor on the machine. Too slow for calling once per frame, fine
// for on-demand lookups.
//
// Non-root degrades instead of failing, verified in that container: exit code 0,
// stderr empty thanks to "-w", and exactly one PID reported — our own, its
// connection the right way round. Its peer comes back as a bare address with no
// PID, the listening process being invisible from there.
func GetSocketsByPid() (map[int][]Socket, error) {
	parser := newLsofSocketParser()

	// -n: Don't resolve host names, they are slow and we do our own resolving
	// -P: Don't resolve port numbers, service names like "ipp" for 631 don't
	//   sort numerically
	// -w: Don't warn about processes we aren't allowed to inspect
	// -i: List internet sockets only. On macOS this reports ICMP sockets too,
	//   which parseLine() drops.
	// -F pfnPT0: Machine readable output with NUL terminated PID, file
	//   descriptor, protocol, name and TCP state fields
	//
	// A plain "-i" rather than "-iTCP -iUDP", which is the narrower spelling and
	// would keep those ICMP records out to begin with: the pair costs more than it
	// buys. Each "-i" is a search item, and lsof exits 1 for every item that
	// located nothing however well the others did, so the pair fails on any machine
	// holding no socket of one kind — a container with an empty /proc/net/tcp fails
	// it always. With one UDP socket up and no TCP, lsof 4.99.4 printed the socket
	// and still exited 1, which "-V" spelled out as "lsof: Internet address not
	// located: TCP". One item makes a non-zero exit mean "no internet sockets at
	// all", which is what the error handling below rests on, and the ICMP records
	// get dropped in parseLine() instead. An fd selection like the "-d cwd" in
	// cwds.go is not a search item and never exits this way.
	//
	// "-Ts" would ask for the TCP state alone and does not narrow the field down,
	// also verified: the queue sizes come along regardless, which is why
	// parseField() dispatches on the "ST=" value prefix.
	commandline := []string{"lsof", "-n", "-P", "-w", "-i", "-F", "pfnPT0"}

	// Locale intentionally left alone, matching GetCwdsByPid()
	err := util.ExecInUsersLocale(commandline, parser.parseLine)
	if err == nil {
		return parser.socketsByPid, nil
	}

	// lsof exits non-zero over an idle machine having no internet socket to
	// report, and an empty listing is the right answer there. The cost is that
	// an lsof failing in no other way passes for that too.
	if !util.IsExitStatus(err) && len(parser.socketsByPid) == 0 {
		// Something other than a non-zero exit code from lsof, this is a real
		// problem.
		return nil, err
	}

	log.Infof("Kept %d processes' worth of sockets despite: %v",
		len(parser.socketsByPid), err)

	return parser.socketsByPid, nil
}

// Parses the output of "lsof -n -P -w -i -F pfnPT0", which comes in NUL
// terminated fields, one line per process and then one line per socket:
//
//	p7619\0
//	f31\0PTCP\0n192.168.50.32:57759->172.217.19.234:443\0TST=ESTABLISHED\0TQR=0\0TQS=0\0
//	f36\0PUDP\0n*:65330\0
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

	if socket.Protocol != "" && socket.Protocol != ProtocolTcp && socket.Protocol != ProtocolUdp {
		// Something beyond the two transport protocols, macOS' ICMP sockets being
		// the ones we know of. Named "*:*" and carrying no port, no peer and no
		// state, so there is no connection to be made of one.
		//
		// Sockets lsof named no protocol for are kept: an lsof that stopped
		// reporting the field should cost us directions, not the whole listing.
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

	case 'P':
		socket.Protocol = Protocol(strings.ToLower(value))

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
