package processes

import (
	"fmt"
	"sort"
	"strconv"
	"time"
)

// Keeps track of all known processes, alive or now dead, and their names.
//
// If multiple process share the same name, the deduplicator can suggest a
// disambiguator string to disambiguate them.
type deduplicator struct {
	// Candidate canonical processes by PID. There can be multiple entries if a
	// PID has been reused over time.
	seenByPid map[int][]*Process

	// Map from command name to list of processes with that name
	byName map[string][]*Process
}

func startTimeDistance(a, b time.Time) time.Duration {
	delta := a.Sub(b)
	if delta < 0 {
		return -delta
	}

	return delta
}

func (d *deduplicator) canonicalProcess(proc *Process) *Process {
	candidates := d.seenByPid[proc.Pid]
	if len(candidates) == 0 {
		return nil
	}

	var closest *Process
	var closestDistance time.Duration
	for _, candidate := range candidates {
		if !candidate.SameAs(proc) {
			continue
		}

		distance := startTimeDistance(candidate.startTime, proc.startTime)
		if closest == nil || distance < closestDistance {
			closest = candidate
			closestDistance = distance
		}
	}

	return closest
}

// Register a process as known. If it was alread known, the registration is a
// no-op.
//
// Processes are considered the same if they have the same PID and the same
// start time.
func (d *deduplicator) register(proc *Process) {
	// Lazy initialization
	if d.seenByPid == nil {
		d.seenByPid = make(map[int][]*Process)
		d.byName = make(map[string][]*Process)
	}

	canonical := d.canonicalProcess(proc)
	if canonical != nil {
		proc.startTime = canonical.startTime
		command := proc.Command()

		// Known process. But does it have the same name as before?

		for _, p := range d.byName[command] {
			if p.SameAs(proc) {
				// Yup, same name as before, registration done
				return
			}
		}

		// But it seems to have changed its name! Processes do that
		// sometimes, through exec() and other means.
		d.byName[command] = append(d.byName[command], proc)

		return
	}

	// Register the process
	command := proc.Command()
	d.seenByPid[proc.Pid] = append(d.seenByPid[proc.Pid], proc)
	d.byName[command] = append(d.byName[command], proc)
}

// Suggest a disambiguating string for this process. Example return values could be:
// - "" (the empty string) if there is only one of these
// - "1" if there are multiple ones with this name, and this is the oldest one
func (d *deduplicator) disambiguator(proc *Process) string {
	command := proc.Command()
	processes := d.byName[command]

	// If only one process with this name, no disambiguation needed
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
		if p.SameAs(proc) {
			return strconv.Itoa(i + 1)
		}
	}

	panic(fmt.Sprintf(
		"PID %d started at %s not found in deduplicator list for command %s: %#v",
		proc.Pid,
		proc.startTime.String(),
		command,
		sorted,
	))
}
