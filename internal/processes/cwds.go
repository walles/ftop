package processes

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/util"
)

// Maps PIDs to the corresponding processes' current working directories.
//
// The listing is partial: processes we aren't allowed to inspect are missing
// from the map, so always check whether a PID is in there before using the
// result. Since other users' processes are usually off limits, expect to only
// get a fraction of the running processes when not running as root.
//
// This forks lsof, which takes a fraction of a second. Too slow for calling
// once per frame, fine for on-demand lookups.
func GetCwdsByPid() (map[int]string, error) {
	parser := newLsofCwdParser()

	// -n: Don't resolve host names, we don't need them and they are slow
	// -w: Don't warn about processes we aren't allowed to inspect
	// -d cwd: List working directories only, this is much faster than listing
	//   every open file of every process
	// -F pfn0: Machine readable output with NUL terminated PID, file
	//   descriptor and name fields
	commandline := []string{"lsof", "-n", "-w", "-d", "cwd", "-F", "pfn0"}

	// Locale intentionally left alone, otherwise lsof renders "Räksmörgås" as
	// "R\xc3\xa4ksm\xc3\xb6rg\xc3\xa5s"
	err := util.ExecInUsersLocale(commandline, parser.parseLine)
	if err == nil {
		return parser.cwds, nil
	}

	// lsof exits non-zero as soon as anything at all went wrong, and failing to
	// inspect some process is business as usual. Whatever it did manage to
	// report is still good, so only give up if we got nothing.
	if len(parser.cwds) == 0 {
		return nil, err
	}

	log.Infof("Listing working directories partially failed, got %d of them: %v", len(parser.cwds), err)

	return parser.cwds, nil
}

// Parses the output of "lsof -d cwd -F pfn0", which comes in NUL terminated
// fields, two newline separated records per process:
//
//	p4542\0
//	fcwd\0n/Users/johan/src/ftop\0
type lsofCwdParser struct {
	cwds map[int]string

	// PID from the most recent "p" field, or -1 before the first one
	pid int

	// True if the most recent "f" field said "cwd"
	isCwd bool
}

func newLsofCwdParser() lsofCwdParser {
	return lsofCwdParser{
		cwds: map[int]string{},
		pid:  -1,
	}
}

// Note that lsof escapes non-printable characters, newlines included, so one
// line of output is always one complete record.
func (parser *lsofCwdParser) parseLine(line string) error {
	for field := range strings.SplitSeq(strings.TrimSuffix(line, "\x00"), "\x00") {
		err := parser.parseField(field)
		if err != nil {
			return err
		}
	}

	return nil
}

func (parser *lsofCwdParser) parseField(field string) error {
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
		parser.isCwd = value == "cwd"

	case 'n':
		if !parser.isCwd {
			// Some other kind of file, we only care about working directories
			return nil
		}

		if value == "" {
			// lsof can report files it has no name for, those tell us nothing
			return nil
		}

		if parser.pid == -1 {
			return fmt.Errorf("lsof reported working directory <%s> before any PID", value)
		}

		parser.cwds[parser.pid] = value
	}

	// lsof can emit fields we didn't ask for, just ignore those
	return nil
}
