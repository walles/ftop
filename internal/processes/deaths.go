package processes

import (
	"time"

	"github.com/walles/ftop/internal/log"
)

// Copy over deadChildrenBirthTimes from the previous frame to the current frame,
// filtering out entries that are too old.
func preserveDeadChildren(baseline, current map[int]*Process) {
	now := time.Now()
	for pid, oldProc := range baseline {
		newProc := current[pid]
		if newProc == nil || !newProc.startTime.Equal(oldProc.startTime) {
			// Process died or PID was reused
			continue
		}

		// Copy over dead children that are still young enough
		for _, birthTime := range oldProc.deadChildrenBirthTimes {
			age := now.Sub(birthTime)
			if age > NATIVITY_MAX_AGE {
				continue
			}

			newProc.deadChildrenBirthTimes = append(newProc.deadChildrenBirthTimes, birthTime)
		}
	}
}

// Track which processes died between baseline and current, and remember their
// launch times.
func trackDeaths(baseline, current map[int]*Process) {
	for pid, oldProc := range baseline {
		newProc := current[pid]
		if newProc != nil && newProc.startTime.Equal(oldProc.startTime) {
			// Same PID, same start time => same process is still alive
			continue
		}

		// This process died

		if oldProc.parent == nil {
			log.Infof("Parent-less process died: %s", oldProc.String())
			continue
		}

		// Find the parent in the current process map
		currentParent := current[oldProc.parent.Pid]
		if currentParent == nil {
			continue
		}

		currentParent.deadChildrenBirthTimes = append(currentParent.deadChildrenBirthTimes, oldProc.startTime)
	}
}
