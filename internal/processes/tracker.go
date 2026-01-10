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
	launches *LaunchNode

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
		log.Errorf("%v", err)
		return
	}

	procsMap := make(map[int]*Process)
	for _, p := range procs {
		procsMap[p.Pid] = p
	}

	tracker.mutex.Lock()

	if tracker.current != nil {
		// Update launch counts tree
		tracker.launches = updateLaunches(tracker.launches, tracker.current, procsMap)
	}

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

func (tracker *Tracker) Processes() []Process {
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

func (tracker *Tracker) Launches() *LaunchNode {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	// Make a copy of our launches tree so that the caller can traverse it
	// without holing or lock.
	if tracker.launches == nil {
		return nil
	}

	var clone func(source *LaunchNode) *LaunchNode
	clone = func(source *LaunchNode) *LaunchNode {
		cloned := &LaunchNode{
			Command:     source.Command,
			LaunchCount: source.LaunchCount,
		}

		if len(source.Children) > 0 {
			cloned.Children = make([]*LaunchNode, 0, len(source.Children))
			for _, ch := range source.Children {
				cloned.Children = append(cloned.Children, clone(ch))
			}
		}
		return cloned
	}

	return clone(tracker.launches)
}
