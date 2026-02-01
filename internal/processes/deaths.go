package processes

import "github.com/walles/ftop/internal/log"

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

		oldProc.parent.deadChildrenBirthTimes = append(oldProc.parent.deadChildrenBirthTimes, oldProc.startTime)
	}
}
