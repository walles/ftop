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

func cmdlineToSlice(cmdline string) []string {
	return cmdlineToSliceWithExists(cmdline, func(path string) bool {
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
	})
}

// This is the testable version of cmdlineToSlice().
//
// The exists function is called to check if a path exists on the filesystem.
func cmdlineToSliceWithExists(cmdline string, exists func(string) bool) []string {
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

	command := filepath.Base(cmdlineToSlice(cmdline)[0])

	// FIXME: Do VM specific parsing here

	return command
}
