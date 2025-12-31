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

func (tracker *Tracker) GetProcesses() []Process {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	procs := make([]Process, 0, len(tracker.current))
	for _, p := range tracker.current {
		proc := *p
		if tracker.baseline == nil {
			// No baseline yet, all processes are new
			zero := time.Duration(0)
			proc.cpuTime = &zero
		} else {
			// For processes already running when we launched, report their
			// times relative to our start time.
			baseProc, ok := tracker.baseline[proc.pid]

			// The start time check protects against reused PIDs. If the start
			// times are different, then they are different processes.
			if ok && proc.startTime.Equal(baseProc.startTime) && proc.cpuTime != nil && baseProc.cpuTime != nil {
				adjusted := *proc.cpuTime - *baseProc.cpuTime
				proc.cpuTime = &adjusted
			}
		}
		procs = append(procs, proc)
	}
	return procs
}
