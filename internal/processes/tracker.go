package processes

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/walles/ptop/internal/log"
)

type Tracker struct {
	mutex    sync.Mutex
	baseline map[int]*Process
	current  map[int]*Process

	OnUpdate chan struct{} // Call GetProcesses() to get the updated list
}

func NewTracker() *Tracker {
	tracker := &Tracker{}
	tracker.OnUpdate = make(chan struct{}, 1)

	// Periodically update the process list
	go func() {
		defer func() {
			log.PanicHandler("process tracker", recover(), debug.Stack())
		}()
		tracker.update() // Initial update
		for range time.NewTicker(time.Second).C {
			tracker.update()
		}
	}()

	return tracker
}

func (tracker *Tracker) update() {
	procs, err := GetAll()
	if err != nil {
		log.Errorf("failed to get process list: %v", err)
		return
	}

	procsMap := make(map[int]*Process)
	for _, p := range procs {
		procsMap[p.Pid] = p
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
			proc.CpuTime = &zero
		} else {
			// For processes already running when we launched, report their
			// times relative to our start time.
			baseProc, ok := tracker.baseline[proc.Pid]

			// The start time check protects against reused PIDs. If the start
			// times are different, then they are different processes.
			if ok && proc.startTime.Equal(baseProc.startTime) && proc.CpuTime != nil && baseProc.CpuTime != nil {
				adjusted := *proc.CpuTime - *baseProc.CpuTime
				proc.CpuTime = &adjusted
			}
		}
		procs = append(procs, proc)
	}
	return procs
}
