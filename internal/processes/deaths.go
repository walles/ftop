package processes

import (
	"time"

	"github.com/walles/ftop/internal/log"
)

// Copy over deadChildrenBirthTimes from the previous frame to the current frame,
// filtering out entries that are too old.
func preserveDeadChildren(matching ProcessMatching) {
	now := time.Now()
	for _, match := range matching.Matched {
		// Copy over dead children that are still young enough
		for _, birthTime := range match.Old.deadChildrenBirthTimes {
			age := now.Sub(birthTime)
			if age > NATIVITY_MAX_AGE {
				continue
			}

			match.New.deadChildrenBirthTimes = append(match.New.deadChildrenBirthTimes, birthTime)
		}
	}
}

// Track which processes died between baseline and current, and remember their
// launch times.
func trackDeaths(matching ProcessMatching) {
	for _, deadProc := range matching.Gone {
		if deadProc.parent == nil {
			log.Infof("Parent-less process died: %s", deadProc.String())
			continue
		}

		currentParent, found := matching.CurrentByPid[deadProc.parent.Pid]
		if !found {
			continue
		}

		currentParent.deadChildrenBirthTimes = append(currentParent.deadChildrenBirthTimes, deadProc.startTime)
	}
}
