package log

import (
	"fmt"
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

func String() string {
	lock.Lock()
	defer lock.Unlock()

	s := "Log entries:\n"
	for _, e := range entries {
		s += fmt.Sprintf("[%s] %s: %s\n", e.timestamp.Format(time.RFC3339), levelToString(e.level), e.message)
	}
	return s
}

func levelToString(level LogLevel) string {
	switch level {
	case LogLevelInfo:
		return "INFO"
	case LogLevelError:
		return "ERROR"
	default:
		panic(fmt.Sprintf("unknown log level: %d", level))
	}
}
