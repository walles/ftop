package processes

import (
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

	// lsof's device column, an opaque string rather than a number because the
	// platforms don't agree on what kind of number it is: macOS gives the kernel
	// address of this very end, "0x77046c8deffe9dd1", while Linux gives the
	// major and minor numbers of the file system a FIFO lives on. Empty wherever
	// lsof reports no device at all, which is macOS for a named FIFO.
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
	// -F pfatdin0: Machine readable output with NUL terminated PID, file
	//   descriptor, access mode, type, device, inode and name fields
	//
	// No filter flag, lsof having none for pipes, so this lists every open file
	// of every process. See GetSocketsByPid() for the cheaper filtered listing
	// the socket sections use.
	commandline := []string{"lsof", "-n", "-w", "-F", "pfatdin0"}

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

// Parses the output of "lsof -n -w -F pfatdin0".
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

func (parser *lsofPipeParser) parseLine(line string) error {
	// TODO: Implement
	return nil
}
