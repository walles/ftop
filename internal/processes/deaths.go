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
		if newProc == nil || !oldProc.SameAs(newProc) {
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
		if newProc != nil && oldProc.SameAs(newProc) {
			// Same PID, same start time => same process is still alive
			continue
		}

		// This process died (or the PID still exists but SameAs() disagrees on
		// start time)

		if newProc != nil {
			// PID still exists but start times don't match — SameAs() returned
			// false. Log details at debug level to help diagnose whether this
			// is a real start-time calculation artifact or an actual SameAs()
			// bug.
			//
			// Reusing PIDs should be rare, let's log on info level. If they
			// turn out to be common, let's re-evaluate how we log this.
			log.Infof(
				"SameAs() mismatch old=%s new=%s old startTime=%s new startTime=%s delta=%s",
				oldProc.String(),
				newProc.String(),
				oldProc.startTime.Format(time.RFC3339Nano),
				newProc.startTime.Format(time.RFC3339Nano),
				newProc.startTime.Sub(oldProc.startTime).Abs(),
			)
		}

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
