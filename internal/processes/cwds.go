package processes

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/walles/ftop/internal/util"
)

// Maps PIDs to the corresponding processes' current working directories.
//
// Processes we aren't allowed to inspect are missing from the map, so always
// check whether a PID is in there before using the result.
//
// This forks lsof, which takes a fraction of a second. Too slow for calling
// once per frame, fine for on-demand lookups.
func GetCwdsByPid() (map[int]string, error) {
	parser := lsofCwdParser{cwds: map[int]string{}}

	// -n: Don't resolve host names, we don't need them and they are slow
	// -w: Don't warn about processes we aren't allowed to inspect
	// -d cwd: List working directories only, this is much faster than listing
	//   every open file of every process
	// -F pfn0: Machine readable output with NUL terminated PID, file
	//   descriptor and name fields
	commandline := []string{"lsof", "-n", "-w", "-d", "cwd", "-F", "pfn0"}
	err := util.Exec(commandline, parser.parseLine)
	if err != nil {
		return nil, err
	}

	return parser.cwds, nil
}

// Parses the output of "lsof -d cwd -F pfn0", which comes in NUL terminated
// fields, two newline separated records per process:
//
//	p4542\0
//	fcwd\0n/Users/johan/src/ftop\0
type lsofCwdParser struct {
	cwds map[int]string

	// PID from the most recent "p" field
	pid int

	// True if the most recent "f" field said "cwd"
	isCwd bool

	// A record whose directory name contains a newline, awaiting its remainder
	partialRecord string
}

func (parser *lsofCwdParser) parseLine(line string) error {
	if parser.partialRecord != "" {
		line = parser.partialRecord + "\n" + line
		parser.partialRecord = ""
	}

	if !strings.HasSuffix(line, "\x00") {
		// Directory names can contain newlines, in which case util.Exec hands
		// us one record in several pieces. Wait for the rest.
		parser.partialRecord = line
		return nil
	}

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
		if parser.pid == 0 {
			return fmt.Errorf("lsof reported working directory <%s> before any PID", value)
		}
		parser.cwds[parser.pid] = value
	}

	// lsof can emit fields we didn't ask for, just ignore those
	return nil
}
