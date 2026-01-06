package log

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	// If we crash, this should go into the crash report
	LogLevelInfo LogLevel = iota

	// This has to be shown to the user after exit, and preferably reported. May
	// contain panic backtraces.
	LogLevelError
)

type entry struct {
	level     LogLevel
	timestamp time.Time
	message   string
}

var entries []entry
var lock = sync.Mutex{}

// If we crash, these messages will go into the crash report
func Infof(format string, args ...any) {
	lock.Lock()
	defer lock.Unlock()

	entries = append(entries, entry{
		level:     LogLevelInfo,
		timestamp: time.Now(),
		message:   fmt.Sprintf(format, args...),
	})
}

// These messages have to be shown to the user after exit, and preferably
// reported.
func Errorf(format string, args ...any) {
	lock.Lock()
	defer lock.Unlock()

	entries = append(entries, entry{
		level:     LogLevelError,
		timestamp: time.Now(),
		message:   fmt.Sprintf(format, args...),
	})
}

func HasErrors() bool {
	lock.Lock()
	defer lock.Unlock()

	for _, e := range entries {
		if e.level == LogLevelError {
			return true
		}
	}
	return false
}

// With highlighted set to true the returned sring may come with ANSI formatting
func String(highlighted bool) string {
	lock.Lock()
	defer lock.Unlock()

	s := "Log entries:\n"
	for _, e := range entries {
		s += fmt.Sprintf(
			"[%s] %s: %s\n",
			e.timestamp.Format(time.RFC3339),
			levelToString(e.level, highlighted),
			formatMessage(e.level, e.message, highlighted),
		)
	}

	return strings.TrimRight(s, "\n")
}

func levelToString(level LogLevel, highlighted bool) string {
	switch level {
	case LogLevelInfo:
		return "INFO"
	case LogLevelError:
		if highlighted {
			// Bold text
			return "\033[1mERROR\033[0m"
		} else {
			return "ERROR"
		}
	default:
		panic(fmt.Sprintf("unknown log level: %d", level))
	}
}

func formatMessage(level LogLevel, message string, highlighted bool) string {
	if !highlighted {
		return message
	}

	if level != LogLevelError {
		return message
	}

	if !strings.Contains(message, "\npanic(") {
		// Not a panic backtrace, no highlighting
		return message
	}

	if !strings.Contains(message, "\n") {
		// Single-line message, not a panic backtrace, no highlighting
		return message
	}

	lines := strings.Split(message, "\n")
	var result strings.Builder

	dim := "\033[2m"
	bold := "\033[1m"
	normal := "\033[0m"

	// If the first line has "crashed:", bold the rest of that line.
	firstLine := lines[0]
	if strings.Contains(firstLine, "crashed:") {
		// Bold the part of the line saying why we crashed
		parts := strings.SplitN(firstLine, "crashed:", 2)
		result.WriteString(parts[0] + "crashed:" + bold + parts[1] + normal + "\n")
	} else {
		result.WriteString(firstLine + "\n")
	}

	// Dim all following lines, up to and including one starting with
	// "\truntime/panic.go". Then bold two lines, then normal from there.
	stage := "dimming"
	for i := 1; i < len(lines); i++ {
		if stage == "dimming" {
			result.WriteString(dim + lines[i] + normal + "\n")
			if strings.HasPrefix(lines[i], "\truntime/panic.go") {
				stage = "normal1"
			}
			continue
		}

		if stage == "normal1" {
			result.WriteString(dimParameters(lines[i]) + "\n")
			stage = "bolding"
			continue
		}

		if stage == "bolding" {
			// Bold everything after the last /
			lastSlashIndex := strings.LastIndex(lines[i], "/")
			if lastSlashIndex >= 0 && lastSlashIndex+1 < len(lines[i]) {
				result.WriteString(lines[i][:lastSlashIndex+1] + bold + lines[i][lastSlashIndex+1:] + normal + "\n")
			} else {
				// We are lost, highlighting will probably just break things
				return message
			}
			stage = "normal"
			continue
		}

		if stage != "normal" {
			panic("unknown stage: " + stage)
		}

		// Normal text, no highlighting
		result.WriteString(dimParameters(lines[i]) + "\n")
	}

	if stage != "normal" {
		// We never reached the normal stage, something went wrong. Just return
		// the original message.
		return message
	}

	// Strip the ending newline and return
	return strings.TrimSuffix(result.String(), "\n")
}

// Dim the first opening parenthesis and everything following it
func dimParameters(line string) string {
	openParenIndex := strings.Index(line, "(")
	if openParenIndex < 0 {
		return line
	}

	dim := "\033[2m"
	normal := "\033[0m"

	return line[:openParenIndex] + dim + line[openParenIndex:] + normal
}
