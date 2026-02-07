package processes

import (
	"fmt"
	"sort"
	"strconv"
)

// Keeps track of all known processes, alive or now dead, and their names.
//
// If multiple process share the same name, the deduplicator can suggest a
// suffix to disambiguate them.
type deduplicator struct {
	// Map from unique process identifier (PID-startTime) to process
	seen map[string]*Process

	// Map from command name to list of processes with that name
	byName map[string][]*Process
}

// Register a process as known. If it was alread known, the registration is a
// no-op.
//
// Processes are considered the same if they have the same PID and the same
// start time.
func (d *deduplicator) register(proc *Process) {
	// Lazy initialization
	if d.seen == nil {
		d.seen = make(map[string]*Process)
		d.byName = make(map[string][]*Process)
	}

	// Create unique key from PID and start time
	key := fmt.Sprintf("%d-%d", proc.Pid, proc.startTime.Unix())

	// Check if already registered
	if _, exists := d.seen[key]; exists {
		return
	}

	// Register the process
	d.seen[key] = proc
	d.byName[proc.Command] = append(d.byName[proc.Command], proc)
}

// Suggest a disambiguating string for this process. Example return values could be:
// - "" (the empty string) if there is only one of these
// - "1" if there are multiple ones with this name, and this is the oldest one
func (d *deduplicator) suffix(proc *Process) string {
	processes := d.byName[proc.Command]

	// If only one process with this name, no suffix needed
	if len(processes) <= 1 {
		return ""
	}

	// Sort by start time (oldest first)
	sorted := make([]*Process, len(processes))
	copy(sorted, processes)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].startTime.Before(sorted[j].startTime)
	})

	// Find this process's position in the sorted list
	for i, p := range sorted {
		if p.Pid == proc.Pid && p.startTime.Equal(proc.startTime) {
			return strconv.Itoa(i + 1)
		}
	}

	panic(fmt.Sprintf(
		"PID %d started at %s not found in deduplicator list for command %s: %#v",
		proc.Pid,
		proc.startTime.String(),
		proc.Command,
		sorted,
	))
}
