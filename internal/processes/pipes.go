package processes

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/util"
)

// How a pipe end is open, as lsof spells it.
//
// Empty when lsof won't say, which is every anonymous pipe on macOS.
type PipeAccess string

const (
	PipeAccessRead      PipeAccess = "r"
	PipeAccessWrite     PipeAccess = "w"
	PipeAccessReadWrite PipeAccess = "u"
	PipeAccessUnknown   PipeAccess = ""
)

// One end of a pipe held open by some process, as reported by lsof.
//
// Which fields are populated is a platform difference, and the two ways of
// matching a pair of ends are built on exactly that, see arePipeEnds().
type PipeEnd struct {
	// lsof's file descriptor number, "1" or similar. Not an identity: one end is
	// reported once per descriptor it is open on. See deduplicatePipeEnds() for
	// what does identify one.
	Fd string

	Access PipeAccess

	// lsof's lowercase "d" column, the kernel address of this very end,
	// "0x77046c8deffe9dd1". A string rather than a number because it is compared
	// and never counted with.
	//
	// Populated for a macOS anonymous pipe and for nothing else: every FIFO record
	// has it empty, on both platforms, so a set Device amounts to a statement that
	// macOS reported this end. The file system device Linux reports for a pipe is a
	// different field, see FileSystemDevice.
	Device string

	// The Device of the end at the other side of this pipe, which is how macOS
	// names an anonymous pipe end. Empty for a pipe lsof doesn't name that way,
	// every Linux one included, and empty on macOS once the other end is gone.
	PeerDevice string

	// The pipe's inode number, the same for both of its ends. A string for the
	// same reason Device is one: it is compared and never counted with. Empty on
	// macOS for an anonymous pipe, and populated on both platforms for a named
	// FIFO.
	Inode string

	// The number of the file system the pipe lives on, which is what tells two
	// pipes sharing an inode number apart. Hex, "0x37", and opaque: it is
	// compared and never counted with.
	//
	// Linux reports it for every pipe, an anonymous one living on pipefs and so
	// sharing it with every other anonymous pipe on the machine. macOS reports it
	// for no pipe at all, neither anonymous nor named, so there two pipes are
	// told apart by their inodes and their kernel addresses alone.
	FileSystemDevice string
}

// Maps PIDs to the pipe ends held open by the corresponding processes, named
// FIFOs included.
//
// The listing is partial: processes we aren't allowed to inspect are missing
// from the map, so expect only a fraction of the running processes when not
// running as root. Processes without any pipes are missing as well.
//
// This forks lsof without a filter, since lsof has no flag for selecting pipes,
// so it costs about twice what the filtered socket listing does. Too slow for
// calling once per frame, fine for on-demand lookups.
func GetPipeEndsByPid() (map[int][]PipeEnd, error) {
	parser := newLsofPipeParser()

	// -n: Don't resolve host names, they are slow and the network sockets this
	//   unfiltered listing drags along would otherwise be looked up one by one
	// -w: Don't warn about processes we aren't allowed to inspect
	// -F pfatdDin0: Machine readable output with NUL terminated PID, file
	//   descriptor, access mode, type, device, file system device, inode and name
	//   fields. The two device fields are different things and we want both: "d"
	//   is lsof's device character code, which on macOS is a pipe end's kernel
	//   address, while "D" is the device number of the file system the file lives
	//   on, which is what tells two FIFOs sharing an inode number apart.
	//
	// No filter flag, lsof having none for pipes, so this lists every open file
	// of every process. See GetSocketsByPid() for the cheaper filtered listing
	// the socket sections use.
	commandline := []string{"lsof", "-n", "-w", "-F", "pfatdDin0"}

	// Locale intentionally left alone, matching GetCwdsByPid()
	err := util.ExecInUsersLocale(commandline, parser.parseLine)
	if err == nil {
		return parser.pipeEndsByPid, nil
	}

	// lsof exits non-zero as soon as anything at all went wrong, and failing to
	// inspect some process is business as usual. Whatever it did manage to
	// report is still good, so only give up if we got nothing.
	if len(parser.pipeEndsByPid) == 0 {
		return nil, err
	}

	log.Infof("Listing pipes partially failed, got %d processes' worth: %v",
		len(parser.pipeEndsByPid), err)

	return parser.pipeEndsByPid, nil
}

// Parses the output of "lsof -n -w -F pfatdDin0", which comes in NUL terminated
// fields, one line per process and then one line per open file:
//
//	p36143\0
//	f1\0a \0tPIPE\0d0x77046c8deffe9dd1\0n->0x652aa8d44c539286\0
//	f4\0ar\0tFIFO\0i82144503\0n/private/tmp/probe.fifo\0
//	f3\0au\0tFIFO\0D0x37\0i2\0n/mnt/a/f\0
//
// The first two lines are macOS, the third Linux, which is the only platform to
// report a "D" field for a pipe at all.
//
// This listing is unfiltered, lsof having no flag for selecting pipes, so most
// of what arrives here is files of other kinds and gets dropped.
type lsofPipeParser struct {
	pipeEndsByPid map[int][]PipeEnd

	// PID from the most recent "p" field, or -1 before the first one
	pid int
}

func newLsofPipeParser() lsofPipeParser {
	return lsofPipeParser{
		pipeEndsByPid: map[int][]PipeEnd{},
		pid:           -1,
	}
}

// Note that lsof escapes non-printable characters, newlines included, so one
// line of output is always one complete record.
func (parser *lsofPipeParser) parseLine(line string) error {
	// Filled in by the fields of this line, whatever kind of file it describes
	var record lsofFileRecord

	for field := range strings.SplitSeq(strings.TrimSuffix(line, "\x00"), "\x00") {
		err := parser.parseField(field, &record)
		if err != nil {
			return err
		}
	}

	if record.fileType != "PIPE" && record.fileType != "FIFO" {
		// A line naming a process, or a file of some other kind, which is most
		// of them in an unfiltered listing. macOS names a unix domain socket
		// exactly the way it names a pipe, so the type is what keeps those two
		// apart.
		return nil
	}

	if parser.pid == -1 {
		return fmt.Errorf("lsof reported pipe on fd <%s> before any PID", record.end.Fd)
	}

	parser.pipeEndsByPid[parser.pid] = append(parser.pipeEndsByPid[parser.pid], record.end)

	return nil
}

// One lsof record being decoded field by field, whether or not it turns out to
// describe a pipe.
type lsofFileRecord struct {
	end PipeEnd

	// lsof's type column: "PIPE", "FIFO", "REG", "IPv4" and so on
	fileType string
}

// Applies one field to record, or to the parser itself for the PID field.
func (parser *lsofPipeParser) parseField(field string, record *lsofFileRecord) error {
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
		record.end.Fd = value

	case 'a':
		record.end.Access = parsePipeAccess(value)

	case 't':
		record.fileType = value

	case 'd':
		record.end.Device = value

	case 'D':
		record.end.FileSystemDevice = value

	case 'i':
		record.end.Inode = value

	case 'n':
		// macOS names an anonymous pipe end by its peer's kernel address, and
		// that is the one name here that identifies anything. Linux calls every
		// anonymous pipe "pipe", a named FIFO is named by its path, and neither
		// one says who is at the other end.
		peerDevice, namesAPeer := strings.CutPrefix(value, "->")
		if namesAPeer {
			record.end.PeerDevice = peerDevice
		}
	}

	// lsof can emit fields we didn't ask for, just ignore those
	return nil
}

// lsof's access mode column, which is a space wherever lsof has nothing to say
// and is that for every anonymous pipe on macOS.
func parsePipeAccess(value string) PipeAccess {
	switch value {
	case "r":
		return PipeAccessRead

	case "w":
		return PipeAccessWrite

	case "u":
		return PipeAccessReadWrite

	default:
		return PipeAccessUnknown
	}
}
