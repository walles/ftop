package processes

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/walles/ftop/internal/log"
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

// Returns nil if we failed to figure out the script name
func parsePythonCommand(cmdline string) *string {
	array := cmdlineToSlice(cmdline, exists)

	// Filter empties
	filtered := make([]string, 0, len(array))
	for _, p := range array {
		if p != "" {
			filtered = append(filtered, p)
		}
	}

	python := filepath.Base(filtered[0])
	if len(filtered) == 1 {
		return &python
	}

	// Ignore some switches, list inspired by 'python2.7 --help'
	ignore := map[string]bool{}
	for _, s := range []string{
		"-b", "-bb", "-B", "-d", "-E", "-i", "-I", "-O", "-OO", "-R", "-s", "-S", "-t", "-tt", "-u", "-v", "-Werror", "-x", "-3",
	} {
		ignore[s] = true
	}
	// Drop leading ignored switches
	for len(filtered) > 1 && ignore[filtered[1]] {
		filtered = append(filtered[:1], filtered[2:]...)
	}

	if len(filtered) == 1 {
		return &python
	}

	if len(filtered) > 2 {
		if filtered[1] == "-m" && !strings.HasPrefix(filtered[2], "-") {
			mod := filepath.Base(filtered[2])
			return &mod
		}
	}

	if strings.HasPrefix(filtered[1], "-") {
		return nil
	}

	if filepath.Base(filtered[1]) == "aws" {
		return parseAwsCommand(filtered[1:])
	}

	script := filepath.Base(filtered[1])
	return &script
}

// Extract "aws command subcommand" from a command line starting with "aws"
func parseAwsCommand(args []string) *string {
	result := []string{"aws"}
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "--profile=") {
			continue
		}
		if strings.HasPrefix(arg, "--region=") {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			break
		}
		if strings.ContainsRune(arg, os.PathSeparator) {
			break
		}

		result = append(result, arg)
		if len(result) >= 4 {
			break
		}
	}

	if len(result) == 4 && result[len(result)-1] != "help" {
		// Drop subcommand if it isn't "help"
		result = result[:len(result)-1]
	}

	joined := strings.Join(result, " ")
	return &joined
}

func prettifyFullyQualifiedJavaClass(className string) *string {
	if className == "" {
		return nil
	}

	parts := strings.Split(className, ".")
	if len(parts) == 1 {
		result := parts[len(parts)-1]
		return &result
	}

	if parts[len(parts)-1] == "Main" {
		result := parts[len(parts)-2] + "." + parts[len(parts)-1]
		return &result
	}

	result := parts[len(parts)-1]
	return &result
}

// Returns nil if we failed to figure out a good Java name
func parseJavaCommand(cmdline string) *string {
	array := cmdlineToSlice(cmdline, exists)
	java := filepath.Base(array[0])
	if len(array) == 1 {
		return &java
	}

	state := "skip next"
	for _, component0 := range array {
		component := component0
		if component == "" {
			continue
		}

		if state == "skip next" {
			if strings.HasPrefix(component, "-") {
				return nil
			}
			state = "scanning"
			continue
		}

		if state == "return next" {
			if strings.HasPrefix(component, "-") {
				return nil
			}
			base := filepath.Base(component)
			return &base
		}

		if state == "scanning" {
			if strings.HasPrefix(component, "-Djdk.java.options=") {
				split := strings.SplitN(component, "=", 2)
				if len(split) == 2 {
					component = split[1]
				}
			}

			if strings.HasPrefix(component, "-X") {
				continue
			}
			if strings.HasPrefix(component, "-D") {
				continue
			}
			if strings.HasPrefix(component, "-ea") {
				continue
			}
			if strings.HasPrefix(component, "-da") {
				continue
			}
			if strings.HasPrefix(component, "-agentlib:") {
				continue
			}
			if strings.HasPrefix(component, "-javaagent:") {
				continue
			}
			if strings.HasPrefix(component, "--add-modules=") {
				continue
			}
			if strings.HasPrefix(component, "@") {
				continue
			}
			if component == "--add-modules" {
				state = "skip next"
				continue
			}
			if strings.HasPrefix(component, "--add-opens=") {
				continue
			}
			if component == "--add-opens" {
				state = "skip next"
				continue
			}
			if strings.HasPrefix(component, "--add-exports=") {
				continue
			}
			if component == "--add-exports" {
				state = "skip next"
				continue
			}
			if strings.HasPrefix(component, "--add-reads=") {
				continue
			}
			if component == "--add-reads" {
				state = "skip next"
				continue
			}
			if strings.HasPrefix(component, "--patch-module=") {
				continue
			}
			if component == "--patch-module" {
				state = "skip next"
				continue
			}
			if component == "-server" {
				continue
			}
			if component == "-noverify" {
				continue
			}
			if component == "-cp" || component == "-classpath" {
				state = "skip next"
				continue
			}
			if component == "-jar" {
				state = "return next"
				continue
			}
			if strings.HasPrefix(component, "-") {
				return nil
			}

			return prettifyFullyQualifiedJavaClass(component)
		}

		log.Infof("The Java command line parser should never get here: <%s>", cmdline)
		return nil
	}

	return nil
}

// Returns nil if we failed to figure out the script name
func parseDartCommand(cmdline string) *string {
	array := cmdlineToSlice(cmdline, exists)
	dart := filepath.Base(array[0])
	if len(array) == 1 {
		return &dart
	}

	for _, candidate := range array[1:] {
		if strings.HasPrefix(candidate, "-") {
			continue
		}

		isAllLower := true
		for _, ch := range candidate {
			if ch < 'a' || ch > 'z' {
				isAllLower = false
				break
			}
		}

		if isAllLower {
			pretty := dart + " " + candidate
			return &pretty
		}

		base := filepath.Base(candidate)
		return &base
	}

	return nil
}

// Returns nil if we failed to figure out the script name
func parseGuileCommand(cmdline string) *string {
	array := cmdlineToSlice(cmdline, exists)
	// Remove empties
	filtered := make([]string, 0, len(array))
	for _, s := range array {
		if s != "" {
			filtered = append(filtered, s)
		}
	}

	// Ignore switches
	ignoreSwitches := map[string]bool{
		"-s":                   true,
		"--listen":             true,
		"-ds":                  true,
		"--debug":              true,
		"--no-debug":           true,
		"--auto-compile":       true,
		"--fresh-auto-compile": true,
		"--no-auto-compile":    true,
		"-q":                   true,
		"--r6rs":               true,
		"--r7rs":               true,
		"-h":                   true,
		"--help":               true,
		"-v":                   true,
		"--version":            true,
	}
	ignoreArgful := map[string]bool{
		"-L": true,
		"-C": true,
		"-x": true,
		"-l": true,
		"-e": true,
	}

	// Consume leading recognized switches
	for len(filtered) > 1 {
		s := filtered[1]
		if ignoreSwitches[s] || strings.HasPrefix(s, "--language=") || strings.HasPrefix(s, "--listen=") || strings.HasPrefix(s, "--use-srfi=") {
			filtered = append(filtered[:1], filtered[2:]...)
			continue
		}
		if ignoreArgful[s] {
			// Special: -l returns its arg as the script if present
			if s == "-l" && len(filtered) > 2 {
				base := filepath.Base(filtered[2])
				return &base
			}
			if len(filtered) > 2 {
				filtered = append(filtered[:1], filtered[3:]...)
			} else {
				filtered = filtered[:1]
			}
			continue
		}
		break
	}

	if len(filtered) == 1 {
		return nil
	}
	if strings.HasPrefix(filtered[1], "-") {
		return nil
	}
	base := filepath.Base(filtered[1])
	return &base
}

// Generic script VM helper: handles VMs like node, ruby, bash, etc. Returns nil
// if we failed to figure out the script name
//
// ignoreSwitches: switches to ignore that do not take an argument
//
// ignoreSwitchesWithArg: switches to ignore that take an argument. "-I" in this
// list will both "-I" and whatever the next argument is.
func parseGenericScriptCommand(cmdline string, ignoreSwitches []string, ignoreSwitchesWithArg []string) *string {
	// Login shells are also commands, the leading - doesn't help anybody.
	// Ref: https://unix.stackexchange.com/questions/38175/difference-between-login-shell-and-non-login-shell
	array := cmdlineToSlice(strings.TrimPrefix(cmdline, "-"), exists)

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
	haveDashDash := false
	for len(filtered) > 1 {
		candidate := filtered[1]
		if candidate == "--" {
			// "--" means "stop processing switches", so remove -- and don't
			// ignore anything else
			haveDashDash = true
			filtered = append(filtered[:1], filtered[2:]...)
			break
		}

		if eq := strings.Index(candidate, "="); eq != -1 {
			candidate = candidate[:eq]
		}
		if ignore[candidate] {
			// Drop the ignored switch
			filtered = append(filtered[:1], filtered[2:]...)
			continue
		}

		// filtered[0] is the command, so to remove filtered[1] (switch) and
		// filtered[2] (switch arg), filtered must have at least 3 elements.
		// Also, make sure filtered[2] is not another switch, that likely means
		// we are lost.
		if slices.Contains(ignoreSwitchesWithArg, candidate) && len(filtered) >= 3 && !strings.HasPrefix(filtered[2], "-") {
			// Drop the ignored switch and its argument
			filtered = append(filtered[:1], filtered[3:]...)
			continue
		}

		// Switches removed to the best of our ability
		break
	}

	vm := filepath.Base(filtered[0])
	if len(filtered) == 1 {
		return &vm
	}

	if strings.HasPrefix(filtered[1], "-") && !haveDashDash {
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

	subcommand := filtered[2]
	if strings.HasPrefix(subcommand, "-") {
		return &script
	}

	pretty := script + " " + subcommand
	return &pretty
}

func parseGitCommand(cmdline string) *string {
	// Example: "git", "show", "-p"
	array := cmdlineToSlice(cmdline, exists)

	if len(array) == 0 {
		return nil
	}

	command := filepath.Base(array[0])

	if len(array) == 1 {
		return &command
	}

	subCommandIndex := 1
	for subCommandIndex < len(array) {
		if array[subCommandIndex] == "-c" {
			// Skip "-c core.quotepath=false"
			subCommandIndex += 2
			continue
		}

		if strings.HasPrefix(array[subCommandIndex], "-") {
			// Unknown option, give up
			return nil
		}

		// Not a switch this is it!
		break
	}

	if subCommandIndex >= len(array) {
		return &command
	}

	gitCommand := command + " " + array[subCommandIndex]
	return &gitCommand
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
