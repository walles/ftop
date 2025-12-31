package processes

import (
	"sync"
	"time"
)

type Tracker struct {
	mutex    sync.Mutex
	baseline map[int]*Process
	current  map[int]*Process

	OnUpdate chan struct{} // Call GetProcesses() to get the updated list
}

func NewTracker() (*Tracker, error) {
	tracker := &Tracker{}
	tracker.OnUpdate = make(chan struct{}, 1)

	// Periodically update the process list
	go func() {
		tracker.update() // Initial update
		for range time.NewTicker(time.Second).C {
			tracker.update()
		}
	}()

	return tracker, nil
}

func (tracker *Tracker) update() {
	procs, err := GetAll()
	if err != nil {
		// FIXME: How do we log this information to the user?
		return
	}

	procsMap := make(map[int]*Process)
	for _, p := range procs {
		procsMap[p.pid] = p
	}

	tracker.adjustTimesSinceBaseline(procsMap)

	tracker.mutex.Lock()
	if tracker.baseline == nil {
		// First iteration
		tracker.baseline = procsMap
	}
	tracker.current = procsMap
	tracker.mutex.Unlock()

	// Notify asynchronously in case nobody is listening
	select {
	case tracker.OnUpdate <- struct{}{}:
	default:
	}
}

// For processes already running when we launched, adjust their times to be
// relative to our start time.
func (tracker *Tracker) adjustTimesSinceBaseline(procs map[int]*Process) {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	for pid, proc := range procs {
		baseProc, ok := tracker.baseline[pid]
		if !ok {
			// New process, no adjustment needed
			continue
		}

		if !proc.startTime.Equal(baseProc.startTime) {
			// This is a new process reusing an old PID
			continue
		}

		if proc.cpuTime == nil || baseProc.cpuTime == nil {
			continue
		}

		adjusted := *proc.cpuTime - *baseProc.cpuTime
		proc.cpuTime = &adjusted
	}
}

func (tracker *Tracker) GetProcesses() []*Process {
	tracker.mutex.Lock()
	procs := make([]*Process, 0, len(tracker.current))
	for _, p := range tracker.current {
		procs = append(procs, p)
	}
	tracker.mutex.Unlock()
	return procs
}
