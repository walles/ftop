package processes

import (
	"os"
	"path/filepath"
	"strings"
)

// Returns nil if we failed to figure out the actual command being run
func parseSudoCommand(cmdline string) *string {
	withoutSudo, found := strings.CutPrefix(cmdline, "sudo ")
	if !found {
		return nil
	}

	if strings.HasPrefix(withoutSudo, "-") {
		// Give up on options
		return nil
	}

	pretty := "sudo " + cmdlineToCommand(withoutSudo)
	return &pretty
}

// Returns nil if we failed to figure out the script name
func parseDotnetCommand(cmdline string) *string {
	parts := cmdlineToSlice(cmdline, exists)

	filtered := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	command := filepath.Base(filtered[0])

	if len(filtered) == 1 {
		return &command
	}

	if strings.HasPrefix(filtered[1], "-") {
		// Second argument is a switch, we don't do switches
		return nil
	}

	if strings.ContainsRune(filtered[1], os.PathSeparator) {
		base := filepath.Base(filtered[1])
		return &base
	}

	pretty := command + " " + filtered[1]
	return &pretty
}
