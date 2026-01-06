package processes

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/walles/ptop/internal/log"
)

// Match "[kworker/0:0H]", no grouping
var LINUX_KERNEL_PROC = regexp.MustCompile(`^\[[^/ ]+/?[^/ ]+\]$`)

// Match "(python2.7)", no grouping
var OSX_PARENTHESIZED_PROC = regexp.MustCompile(`^\\([^()]+\\)$`)

// Extract a potential file path from the end of a string.
//
// The coalescing logic will then base decisions on whether this file path
// exists or not.
func getTrailingAbsolutePath(partialCmdline string) *string {

	startIndex := -1
	if strings.HasPrefix(partialCmdline, "/") {
		startIndex = 0
	} else if strings.HasPrefix(partialCmdline, "-") {
		equalsSlashIndex := strings.Index(partialCmdline, "=/")
		if equalsSlashIndex == -1 {
			// No =/ in the string, so we're not looking at -Djava.io.tmpdir=/tmp
			return nil
		}

		// Start at the slash after the equals sign
		startIndex = equalsSlashIndex + 1
	}

	if startIndex == -1 {
		// Start not found, this is not a path
		return nil
	}

	colonSlashIndex := strings.LastIndex(partialCmdline[startIndex:], ":/")
	if colonSlashIndex == -1 {
		s := partialCmdline[startIndex:]
		return &s
	}

	// strings.LastIndex() on the sliced string returned an index relative to
	// the slice, so add startIndex to get the index in the original string.
	absolute := partialCmdline[startIndex+colonSlashIndex+1:]
	return &absolute
}

// Two or more (previously) space separated command line parts should be
// coalesced if combining them with a space in between creates an existing file
// path, or a : separated series of file paths.
//
// Return values:
//
//   - true: Coalesce, done
//   - false: Do not coalesce, done
//   - nil: Undecided, add another part and try again
func shouldCoalesce(parts []string, exists func(string) bool) *bool {
	last := parts[len(parts)-1]
	if strings.HasPrefix(last, "-") || strings.HasPrefix(last, "/") {
		// Last part starts a command line option or a new absolute path, don't
		// coalesce.
		res := false
		return &res
	}

	coalesced := strings.Join(parts, " ")
	candidatePtr := getTrailingAbsolutePath(coalesced)
	if candidatePtr == nil {
		// This is not a candidate for coalescing
		res := false
		return &res
	}

	candidate := *candidatePtr
	if exists(candidate) {
		// Found it, done!
		res := true
		return &res
	}

	parent := filepath.Dir(candidate)
	if exists(parent) {
		// Found the parent directory, we're on the right track, keep looking!
		return nil
	}

	// Candidate does not exists, and neither does its parent directory, this is
	// not it.
	res := false
	return &res
}

// How many parts should be coalesced?
func coalesceCount(parts []string, exists func(string) bool) int {
	return 0
}

// Helper function for keeping cmdlineToSlice testable.
func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}

	log.Infof("Failed to check file existance for <%s>: %v", path, err)

	// Who knows what to return here? False is the safe option that prevents
	// coalescing.
	return false
}

// This is the testable version of cmdlineToSlice().
//
// The exists function is called to check if a path exists on the filesystem.
func cmdlineToSlice(cmdline string, exists func(string) bool) []string {
	// FIXME: Handle paths with spaces in them
	return strings.Fields(cmdline)
}

// Extracts the command from the command line.
//
// This function most often returns the first component of the command line with
// the path stripped away.
//
// For some language runtimes, this function may return the name of the program
// that the runtime is executing.
func cmdlineToCommand(cmdline string) string {
	if LINUX_KERNEL_PROC.MatchString(cmdline) {
		return cmdline
	}

	if OSX_PARENTHESIZED_PROC.MatchString(cmdline) {
		return cmdline
	}

	command := filepath.Base(cmdlineToSlice(cmdline, exists)[0])

	// FIXME: Do VM specific parsing here

	return command
}
