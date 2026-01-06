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

// Generic script VM helper: handles VMs like node, ruby, bash, etc.
// Returns nil if we failed to figure out the script name
func parseGenericScriptCommand(cmdline string, ignoreSwitches []string) *string {
	array := cmdlineToSlice(cmdline, exists)

	// Filter empties
	filtered := make([]string, 0, len(array))
	for _, p := range array {
		if p != "" {
			filtered = append(filtered, p)
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	// Remove leading switches that match ignoreSwitches (matching up to '=' )
	ignore := make(map[string]bool, len(ignoreSwitches))
	for _, s := range ignoreSwitches {
		ignore[s] = true
	}
	for len(filtered) > 1 {
		key := filtered[1]
		if eq := strings.Index(key, "="); eq != -1 {
			key = key[:eq]
		}
		if !ignore[key] {
			break
		}
		// Drop the ignored switch
		filtered = append(filtered[:1], filtered[2:]...)
	}

	vm := filepath.Base(filtered[0])
	if len(filtered) == 1 {
		return &vm
	}

	if strings.HasPrefix(filtered[1], "-") {
		// Unknown option, help!
		return nil
	}

	script := filepath.Base(filtered[1])
	if len(filtered) == 2 {
		return &script
	}

	if script != "brew.rb" && script != "brew.sh" && script != "yarn.js" {
		return &script
	}

	sub := filtered[2]
	if strings.HasPrefix(sub, "-") {
		return &script
	}

	pretty := script + " " + sub
	return &pretty
}

// Returns nil if we failed to figure out the subcommand
func parseWithSubcommand(cmdline string, ignoreSwitches []string) *string {
	array := cmdlineToSlice(cmdline, exists)

	// Remove leading switches that match ignoreSwitches (matching up to '=')
	ignore := make(map[string]bool, len(ignoreSwitches))
	for _, s := range ignoreSwitches {
		ignore[s] = true
	}
	for len(array) > 1 {
		key := array[1]
		if eq := strings.Index(key, "="); eq != -1 {
			key = key[:eq]
		}
		if !ignore[key] {
			break
		}
		// Drop the ignored switch
		array = append(array[:1], array[2:]...)
	}

	command := filepath.Base(array[0])
	if len(array) == 1 {
		return &command
	}

	if strings.HasPrefix(array[1], "-") {
		// Unknown option, help!
		return nil
	}

	pretty := command + " " + array[1]
	return &pretty
}

// (Node handling now inlined at call site in commandline.go)
