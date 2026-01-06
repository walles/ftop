package processes

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/walles/ptop/internal/log"
)

// Match "[kworker/0:0H]", no grouping
var LINUX_KERNEL_PROC = regexp.MustCompile(`^\[[^/ ]+/?[^/ ]+\]$`)

// Match "(python2.7)", no grouping
var OSX_PARENTHESIZED_PROC = regexp.MustCompile(`^\\([^()]+\\)$`)

// Name of the Perl interpreter
var PERL_BIN = regexp.MustCompile(`^perl[.0-9]*$`)

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
	for coalesceCount := 2; coalesceCount <= len(parts); coalesceCount++ {
		should := shouldCoalesce(parts[0:coalesceCount], exists)

		if should == nil {
			// Undecided, keep looking
			continue
		}

		if !*should {
			return 1
		}

		// should == true
		return coalesceCount
	}

	// Undecided until the end, this means no coalescing should be done
	return 1
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
	baseSplit := strings.Split(cmdline, " ")
	if len(baseSplit) == 1 {
		return baseSplit
	}

	merged := make([]string, 0, len(baseSplit))
	for i := 0; i < len(baseSplit); {
		cc := coalesceCount(baseSplit[i:], exists)
		merged = append(merged, strings.Join(baseSplit[i:i+cc], " "))
		i += cc
	}

	return merged
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

	// Electron embeds inside .app bundles; clarify to app name
	if command == "Electron" {
		clarified := tryClarifyElectron(cmdline)
		if clarified != nil && *clarified != "" {
			return *clarified
		}
	}
	if strings.HasPrefix(command, "python") || command == "Python" {
		return faillog(cmdline, parsePythonCommand(cmdline))
	}

	if command == "sudo" {
		return faillog(cmdline, parseSudoCommand(cmdline))
	}

	if command == "ruby" {
		return faillog(cmdline, parseGenericScriptCommand(cmdline, []string{
			"-a",
			"-d",
			"--debug",
			"--disable",
			"-Eascii-8bit:ascii-8bit",
			"-l",
			"-n",
			"-p",
			"-s",
			"-S",
			"-v",
			"--verbose",
			"-w",
			"-W0",
			"-W1",
			"-W2",
			"--",
		}))
	}

	if command == "bash" || command == "sh" || command == "zsh" {
		return faillog(cmdline, parseGenericScriptCommand(cmdline, []string{"-p"}))
	}

	if command == "node" {
		return faillog(cmdline, parseGenericScriptCommand(cmdline, []string{
			"--max_old_space_size",
			"--no-warnings",
			"--enable-source-maps",
		}))
	}

	if command == "dotnet" {
		return faillog(cmdline, parseDotnetCommand(cmdline))
	}

	if command == "dart" {
		return faillog(cmdline, parseDartCommand(cmdline))
	}

	if strings.HasPrefix(command, "guile") {
		return faillog(cmdline, parseGuileCommand(cmdline))
	}

	if slices.Contains([]string{
		"apt-get",
		"apt",
		"cargo",
		"docker",
		"docker-compose",
		"git",
		"go",
		"npm",
		"pip",
		"pip3",
		"rustup",
	}, command) {
		return faillog(cmdline, parseWithSubcommand(cmdline, nil))
	}

	if command == "terraform" {
		return faillog(cmdline, parseWithSubcommand(cmdline, []string{"-chdir"}))
	}

	if PERL_BIN.MatchString(command) {
		return faillog(cmdline, parseGenericScriptCommand(cmdline, nil))
	}

	// macOS app / framework prefixing and human-friendly shortening
	appNamePrefix := getAppNamePrefix(cmdline)
	if isHumanFriendly(command) {
		appNamePrefix = ""
	}

	if len(command) < 25 {
		return appNamePrefix + command
	}

	commandSplit := strings.Split(command, ".")
	if len(commandSplit) > 1 {
		commandSuggestion := ""
		last := commandSplit[len(commandSplit)-1]
		if len(last) > 4 {
			commandSuggestion = last
		} else if len(commandSplit) >= 2 {
			commandSuggestion = commandSplit[len(commandSplit)-2]
		}
		if len(commandSuggestion) >= 5 {
			command = commandSuggestion
		}
	}

	return appNamePrefix + command
}

// If successful, just return the result. If unsuccessful log the problem and
// return the VM name.
func faillog(cmdline string, parseResult *string) string {
	if parseResult != nil {
		return *parseResult
	}

	log.Infof("Parsing failed, using fallback: <%s>", cmdline)
	return filepath.Base(cmdlineToSlice(cmdline, exists)[0])
}

// AKA "Does this command contain any capital letters?"
func isHumanFriendly(command string) bool {
	for _, r := range command {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// On macOS, get which app this command is part of
func getAppNamePrefix(cmdline string) string {
	commandWithPath := cmdlineToSlice(cmdline, exists)[0]
	command := filepath.Base(commandWithPath)
	parts := strings.Split(commandWithPath, "/")
	for _, part := range parts {
		if !strings.Contains(part, ".") {
			continue
		}

		idx := strings.LastIndex(part, ".")
		if idx <= 0 || idx+1 >= len(part) {
			continue
		}

		name := part[:idx]
		suffix := part[idx+1:]
		if suffix != "app" && suffix != "framework" {
			continue
		}

		if name == command {
			continue
		}

		return name + "/"
	}

	return ""
}

// If any path component of the command ends with .app, return that component without the suffix
func tryClarifyElectron(cmdline string) *string {
	commandWithPath := cmdlineToSlice(cmdline, exists)[0]
	parts := strings.Split(commandWithPath, "/")
	for _, part := range parts {
		if strings.HasSuffix(part, ".app") {
			name := part[:len(part)-4]
			return &name
		}
	}

	return nil
}
