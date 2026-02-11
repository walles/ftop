package processes

import (
	"strconv"
	"strings"
)

func Filter(processes []Process, filter string) []Process {
	if filter == "" {
		return processes
	}

	filtered := make([]Process, 0, len(processes))
	for _, process := range processes {
		if process.Matches(filter) {
			filtered = append(filtered, process)
		}
	}
	return filtered
}

func (p *Process) Matches(filter string) bool {
	if strings.Contains(p.cmdline, filter) {
		return true
	}

	lowerCaseFilter := strings.ToLower(filter)
	if strings.Contains(strings.ToLower(p.cmdline), lowerCaseFilter) {
		return true
	}
	if strings.Contains(strings.ToLower(p.Command), lowerCaseFilter) {
		return true
	}

	// FIXME: If the filter matches the username exactly, then maybe we should
	// make sure to *not* match other usernames, even if it's a substring of
	// them.
	if strings.Contains(strings.ToLower(p.Username), lowerCaseFilter) {
		return true
	}

	pidStr := strconv.Itoa(p.Pid)
	if strings.Contains(pidStr, filter) { // nolint:S1008
		return true
	}

	return false
}
