package processes

import "strings"

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
