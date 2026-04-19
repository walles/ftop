package processes

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"unicode"

	"github.com/walles/ftop/internal/log"
)

// Match "[kworker/0:0H]", no grouping
var LINUX_KERNEL_PROC = regexp.MustCompile(`^\[[^/ ]+/?[^/ ]+\]$`)

// Match "(python2.7)", no grouping
var OSX_PARENTHESIZED_PROC = regexp.MustCompile(`^\\([^()]+\\)$`)

// Name of the Perl interpreter
var PERL_BIN = regexp.MustCompile(`^perl[.0-9]*$`)

// From command line to command name
var commandCache = make(map[string]string)

type existence byte

const (
	existenceFalse existence = iota
	existenceTrue
	existenceNotYet
	existenceError // Can't tell whether this exists or not
)

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
func shouldCoalesce(parts []string, exists func(string) existence) existence {
	last := parts[len(parts)-1]
	if strings.HasPrefix(last, "-") || strings.HasPrefix(last, "/") {
		// Last part starts a command line option or a new absolute path, don't
		// coalesce.
		return existenceFalse
	}

	coalesced := strings.Join(parts, " ")
	candidatePtr := getTrailingAbsolutePath(coalesced)
	if candidatePtr == nil {
		// This is not a candidate for coalescing
		return existenceFalse
	}

	return exists(*candidatePtr)
}

// How many parts should be coalesced?
func coalesceCount(parts []string, exists func(string) existence) (int, error) {
	for coalesceCount := 2; coalesceCount <= len(parts); coalesceCount++ {
		should := shouldCoalesce(parts[0:coalesceCount], exists)

		if should == existenceNotYet {
			// Undecided, keep looking
			continue
		}

		if should == existenceError {
			return 0, fmt.Errorf("failed to check path existence while slicing command line")
		}

		if should == existenceFalse {
			return 1, nil
		}

		if should != existenceTrue {
			panic(fmt.Errorf("unsupported coalescing state: %v", should))
		}

		return coalesceCount, nil
	}

	// Undecided until the end, this means no coalescing should be done
	return 1, nil
}

func isFileNameTooLong(err error) bool {
	pathErr, ok := err.(*os.PathError)
	if !ok {
		return false
	}

	return pathErr.Err == syscall.ENAMETOOLONG
}

// Helper function for keeping cmdlineToSlice testable
func exists(path string) existence {
	_, err := os.Stat(path)
	if err == nil {
		return existenceTrue
	}

	// Not there, at least not yet

	if isFileNameTooLong(err) {
		// Not a valid file name, back off!
		return existenceFalse
	}

	if !os.IsNotExist(err) {
		// FIXME: Maybe not log permission errors? Or log those as debug but other errors as info?
		log.Debugf("Failed to check file existence: %v", err)
		return existenceError
	}

	// File does not exist, but could it if we got more parts?
	parent := filepath.Dir(path)
	parentExists := exists(parent)
	switch parentExists {
	case existenceTrue:
		// Parent exists, maybe the file will show up if we add more parts
		return existenceNotYet
	case existenceFalse:
		// Parent does not exist, this is not it
		return existenceFalse
	case existenceError:
		// Cannot check parent, propagate the error
		return existenceError
	case existenceNotYet:
		// The parent did not exist
		return existenceFalse
	default:
		panic(fmt.Errorf("unsupported parent existence state: %v", parentExists))
	}
}

// Convert "ls dir/" into ["ls", "dir/"]. Also handle spaces, so it can convert
// "ls My Documents/" into ["ls", "My Documents/"].
//
// The exists function is called to check if a path exists on the filesystem. It
// is a parameter of its own for testability reasons.
func cmdlineToSlice(cmdline string, exists func(string) existence) ([]string, error) {
	baseSplit := strings.Split(cmdline, " ")
	if len(baseSplit) == 1 {
		return baseSplit, nil
	}

	merged := make([]string, 0, len(baseSplit))
	for i := 0; i < len(baseSplit); {
		cc, err := coalesceCount(baseSplit[i:], exists)
		if err != nil {
			// FIXME: Check all callers that they handle this error well
			return nil, err
		}

		merged = append(merged, strings.Join(baseSplit[i:i+cc], " "))
		i += cc
	}

	return merged, nil
}

// Convert a command line string into a command name. Examples:
//   - "ls dir/" -> "ls"
//   - "java -jar myapp.jar" -> "myapp.jar"
//   - "sh -c cd /some/dir && echo hello" -> "echo"
func cmdlineToCommand(cmdline string, pid int) string {
	cached, found := commandCache[cmdline]
	if found {
		return cached
	}

	result := cmdlineToCommandInternal(cmdline, pid)
	commandCache[cmdline] = result
	return result
}

// If cmdline slicing fails, we fall back to ps -o comm= for that PID. And if
// that fails, we fall back to whitespace splitting.
func cmdlineToCommandInternal(cmdline string, pid int) string {
	if LINUX_KERNEL_PROC.MatchString(cmdline) {
		return cmdline
	}

	if OSX_PARENTHESIZED_PROC.MatchString(cmdline) {
		return cmdline
	}

	argv, err := cmdlineToSlice(cmdline, exists)
	if err == nil {
		return argvToCommand(argv)
	}

	log.Infof("Failed to slice command line for command parsing for process %d, falling back to comm=: %v", pid, err)

	executable, executableErr := getExecutableForPid(pid)
	if executableErr == nil {
		return argvToCommand([]string{executable})
	}

	log.Infof("Failed to get comm= for process %d, falling back to space split command parsing: %v, cmdline=<%s>", pid, executableErr, cmdline)

	return argvToCommand(strings.Fields(cmdline))
}

// Convert "sh -c cd /some/dir && echo hello" to just "echo hello".
func stripShellPrefix(argv []string) []string {
	shell := filepath.Base(argv[0])
	if !slices.Contains([]string{"sh", "bash", "zsh", "fish"}, shell) {
		return argv
	}

	// "sh -c cd /some/dir && echo hello"
	if len(argv) >= 6 && argv[1] == "-c" && argv[2] == "cd" && argv[4] == "&&" {
		return argv[5:]
	}

	// "sh -c which minikube"
	if len(argv) >= 3 && argv[1] == "-c" {
		return argv[2:]
	}

	return argv
}

func argvToCommand(argv []string) string {
	if len(argv) == 0 {
		return ""
	}

	argv = stripShellPrefix(argv)
	command := filepath.Base(argv[0])

	// Electron embeds inside .app bundles; clarify to app name
	if command == "Electron" {
		clarified := tryClarifyElectron(argv)
		if clarified != nil && *clarified != "" {
			return *clarified
		}
	}
	if strings.HasPrefix(command, "python") || command == "Python" {
		return faillog(argv, parsePythonCommand(argv))
	}

	if command == "java" {
		return faillog(argv, parseJavaCommand(argv))
	}

	if command == "sudo" {
		return faillog(argv, parseSudoCommand(argv))
	}

	if command == "ruby" {
		return faillog(argv, parseGenericScriptCommand(argv, []string{
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
		}, []string{
			"-I",
		}))
	}

	// Login shells are also commands, the leading - doesn't help anybody.
	// Ref: https://unix.stackexchange.com/questions/38175/difference-between-login-shell-and-non-login-shell
	if slices.Contains([]string{"fish", "bash", "sh", "zsh"}, strings.TrimPrefix(command, "-")) {
		return faillog(argv, parseGenericScriptCommand(argv, []string{"-p", "-l"}, nil))
	}

	if command == "node" {
		return faillog(argv, parseGenericScriptCommand(argv, []string{
			"--max_old_space_size",
			"--no-warnings",
			"--enable-source-maps",
		}, nil))
	}

	if command == "dotnet" {
		return faillog(argv, parseDotnetCommand(argv))
	}

	if command == "dart" {
		return faillog(argv, parseDartCommand(argv))
	}

	if strings.HasPrefix(command, "guile") {
		return faillog(argv, parseGuileCommand(argv))
	}

	if command == "git" {
		return faillog(argv, parseGitCommand(argv))
	}

	if slices.Contains([]string{
		"apt-get",
		"apt",
		"cargo",
		"docker",
		"docker-compose",
		"go",
		"npm",
		"pip",
		"pip3",
		"rustup",
	}, command) {
		return faillog(argv, parseWithSubcommand(argv, nil))
	}

	if command == "which" {
		return faillog(argv, parseWithSubcommand(argv, nil))
	}

	if command == "terraform" {
		return faillog(argv, parseWithSubcommand(argv, []string{"-chdir"}))
	}

	if PERL_BIN.MatchString(command) {
		return faillog(argv, parseGenericScriptCommand(argv, nil, nil))
	}

	// macOS app / framework prefixing and human-friendly shortening
	appNamePrefix := getAppNamePrefix(argv)
	if isHumanFriendly(command) {
		appNamePrefix = ""
	}

	if len(command) < 25 {
		return coalesceAppCommand(appNamePrefix + command)
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

	return coalesceAppCommand(appNamePrefix + command)
}

// Convert "GenerativeExperiencesRuntime/generativeexperiencesd" to
// "GenerativeExperiencesRuntime (Daemon)" based on that both the first and the
// second part have the same prefix, plus "d" is a common suffix for daemons.
func coalesceAppCommand(command string) string {
	if strings.Count(command, "/") != 1 {
		return command
	}

	parts := strings.SplitN(command, "/", 2)
	first := parts[0]
	second := parts[1]

	// Normalize for comparison by removing spaces and dashes
	firstNormalized := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(first), " ", ""), "-", "")
	secondNormalized := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(second), " ", ""), "-", "")

	if !strings.HasSuffix(secondNormalized, "d") {
		return command
	}

	secondWithoutD := secondNormalized[:len(secondNormalized)-1]

	if !strings.HasPrefix(firstNormalized, secondWithoutD) {
		return command
	}

	// Replace dashes with spaces in the output
	firstOutput := strings.ReplaceAll(first, "-", " ")
	return firstOutput + " (Daemon)"
}

// If successful, just return the result. If unsuccessful log the problem and
// return the VM name.
func faillog(argv []string, parseResult *string) string {
	if parseResult != nil {
		return *parseResult
	}

	log.Infof("Parsing failed, using fallback: <%s>", argv)
	return filepath.Base(argv[0])
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
func getAppNamePrefix(argv []string) string {
	commandWithPath := argv[0]
	command := filepath.Base(commandWithPath)
	parts := strings.SplitSeq(commandWithPath, "/")
	for part := range parts {
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
func tryClarifyElectron(argv []string) *string {
	commandWithPath := argv[0]
	parts := strings.Split(commandWithPath, "/")
	for _, part := range parts {
		if strings.HasSuffix(part, ".app") {
			name := part[:len(part)-4]
			return &name
		}
	}

	return nil
}
