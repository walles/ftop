package processes

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/util"
)

// One unix domain socket held open by some process, as reported by lsof.
//
// Which fields are populated says what the socket is: a client's connected
// socket names its peer, a listener and the sockets accepted on it carry the
// path they are bound to, and no socket carries both. Which makes the naming one
// directional for a connection made over a path, and that is what
// UnixSocketConnections() reads a connection's direction off. The two ends of a
// socketpair(2) do name each other, the way a pipe's ends match mutually, and
// that is exactly the case it can draw no arrow for.
type UnixSocket struct {
	// lsof's file descriptor number, "3" or similar. Not an identity: one socket
	// is reported once per descriptor it is open on, and the Device is what
	// identifies it instead.
	Fd string

	// lsof's lowercase "d" column, the kernel address of this very socket,
	// "0xd82e0ac85b8e6a97". A string rather than a number because it is compared
	// and never counted with.
	//
	// Comparing the text is enough, lsof padding neither this nor PeerDevice:
	// 94.7% of distinct devices and 94.8% of distinct peers are 16 hex digits,
	// the same distribution, and of 489 peers measured on a quiet macOS laptop
	// none resolved numerically but not textually. Two hex numbers of different
	// lengths are just two numbers of different magnitudes.
	Device string

	// The Device of the socket at the other end, from lsof's "n->0x..." name.
	// Empty for a socket lsof doesn't name that way, which is every socket
	// carrying a Path, and empty for one whose peer is gone.
	PeerDevice string

	// The file system path this socket is bound to, "/tmp/probe.sock". Carried by
	// a listener and by every socket accepted on it, empty for a socket that
	// dialed one of those and empty for both ends of a socketpair(2).
	Path string
}

// Maps PIDs to the unix domain sockets held open by the corresponding
// processes.
//
// The listing is partial: processes we aren't allowed to inspect are missing
// from the map, so expect only a fraction of the running processes when not
// running as root. Processes without any unix sockets are missing as well.
//
// This forks lsof, which takes a fraction of a second. Too slow for calling
// once per frame, fine for on-demand lookups.
func GetUnixSocketsByPid() (map[int][]UnixSocket, error) {
	parser := newLsofUnixSocketParser()

	// -n: Don't resolve host names. A unix socket has none to resolve, so this
	//   only makes sure lsof never goes looking.
	// -w: Don't warn about processes we aren't allowed to inspect
	// -U: List unix domain sockets only, which is the filter pipes have no
	//   equivalent of. So this listing costs what the "-i" one in sockets.go
	//   does rather than what the unfiltered one in GetPipeEndsByPid() does.
	// -F pfnd0: Machine readable output with NUL terminated PID, file
	//   descriptor, name and device fields. The device is this socket's own
	//   kernel address and the name is the peer's, and that pair is what the peer
	//   matching in UnixSocketConnections() is built on.
	commandline := []string{"lsof", "-n", "-w", "-U", "-F", "pfnd0"}

	// Locale intentionally left alone, matching GetCwdsByPid()
	err := util.ExecInUsersLocale(commandline, parser.parseLine)
	if err == nil {
		return parser.unixSocketsByPid, nil
	}

	// A machine holding no unix socket at all is an empty listing rather than a
	// failure, the way an idle machine is for GetSocketsByPid(), so the exit
	// status alone doesn't fail this the way it does the pipe listing. The cost is
	// that an lsof failing in no other way passes for that too.
	//
	// Defensive rather than known to be needed: "-i" makes lsof exit 1 whenever it
	// locates no internet socket, but "-U" over a container holding no unix socket
	// at all printed nothing and exited 0 on lsof 4.99.4. Whether some other lsof
	// counts "-U" as a search item the way it counts "-i" is untested.
	if !util.IsExitStatus(err) && len(parser.unixSocketsByPid) == 0 {
		// Something other than a non-zero exit code from lsof, this is a real
		// problem.
		return nil, err
	}

	log.Infof("Kept %d processes' worth of unix sockets despite: %v",
		len(parser.unixSocketsByPid), err)

	return parser.unixSocketsByPid, nil
}

// Parses the output of "lsof -n -w -U -F pfnd0", which comes in NUL terminated
// fields, one line per process and then one line per socket:
//
//	p78879\0
//	f3\0d0xe396ab59862314e8\0n/tmp/probe.sock\0
//	f4\0d0xd82e0ac85b8e6a97\0n/tmp/probe.sock\0
//	f0\0d0x520254f296b152e3\0n->(none)\0
//	p78881\0
//	f3\0d0x628a5982efaac095\0n->0xd82e0ac85b8e6a97\0
//
// A server and its client. The server's first two sockets are the listener and
// the socket it accepted the client on, both named by the path they are bound
// to, and its third has lost whoever was at the other end. The client names the
// accepted socket by that socket's device and carries no path.
//
// Those are the only three shapes a name comes in, and a record has a peer or a
// path and never both: measured on a quiet macOS laptop, 551 records, 462 with a
// peer, 82 with a path, none with the two of them.
//
// "-U" selects unix sockets and nothing else, so unlike lsofPipeParser this
// needs no type field to tell its own records apart from the rest.
type lsofUnixSocketParser struct {
	unixSocketsByPid map[int][]UnixSocket

	// PID from the most recent "p" field, or -1 before the first one
	pid int
}

func newLsofUnixSocketParser() lsofUnixSocketParser {
	return lsofUnixSocketParser{
		unixSocketsByPid: map[int][]UnixSocket{},
		pid:              -1,
	}
}

// Note that lsof escapes non-printable characters, newlines included, so one
// line of output is always one complete record.
func (parser *lsofUnixSocketParser) parseLine(line string) error {
	// Filled in by the fields of this line, whether or not it turns out to
	// describe a socket
	var record lsofUnixSocketRecord

	for field := range strings.SplitSeq(strings.TrimSuffix(line, "\x00"), "\x00") {
		err := parser.parseField(field, &record)
		if err != nil {
			return err
		}
	}

	if !record.isASocket {
		// A line naming the process the sockets below it belong to. With "-U"
		// there is nothing else it can be.
		return nil
	}

	if parser.pid == -1 {
		return fmt.Errorf("lsof reported unix socket on fd <%s> before any PID", record.socket.Fd)
	}

	parser.unixSocketsByPid[parser.pid] = append(parser.unixSocketsByPid[parser.pid], record.socket)

	return nil
}

// One lsof record being decoded field by field, whether it describes a socket or
// the process holding one.
type lsofUnixSocketRecord struct {
	socket UnixSocket

	// True once a file descriptor field has arrived, which is what tells a socket
	// apart from the process line above it
	isASocket bool
}

// Applies one field to record, or to the parser itself for the PID field.
func (parser *lsofUnixSocketParser) parseField(field string, record *lsofUnixSocketRecord) error {
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
		record.socket.Fd = value
		record.isASocket = true

	case 'd':
		record.socket.Device = value

	case 'n':
		applyUnixSocketName(value, &record.socket)
	}

	// lsof can emit fields we didn't ask for, just ignore those
	return nil
}

// Applies lsof's name column to socket, which on macOS is one of three things:
// the kernel address of the socket at the other end, spelled "->0x...", the path
// this socket is bound to, or "->(none)" for a socket whose peer is gone.
//
// A name is a peer when it starts with "->" and a path in every other case. Not
// every path is absolute — a socket bound to a relative one is named by just
// that, with no leading slash to recognize it by — so testing for a slash
// instead would drop those.
//
// Linux spells the name a fourth way this makes no sense of, and knowingly:
// "/tmp/probe.sock type=STREAM" for a bound socket and a bare "type=STREAM" for
// one that isn't, verified against a real connected pair on lsof 4.99.4. So a
// Linux Path comes out with that suffix on it, or holding nothing but the suffix.
// Harmless while Linux reports no peer either way, no peer meaning no connection
// and so nothing that ever renders a Path; the slice that gives Linux its peers
// is the one that has to deal with this.
func applyUnixSocketName(name string, socket *UnixSocket) {
	peerDevice, namesAPeer := strings.CutPrefix(name, "->")
	if !namesAPeer {
		socket.Path = name
		return
	}

	if peerDevice == "(none)" {
		// Whoever was at the other end is gone, so there is nobody to name
		return
	}

	socket.PeerDevice = peerDevice
}
