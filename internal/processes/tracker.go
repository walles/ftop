package processes

import (
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/walles/ftop/internal/log"
)

type Tracker struct {
	mutex    sync.Mutex
	baseline map[int]*Process // Previous iteration, for tracking births and deaths
	current  map[int]*Process
	launches *LaunchNode

	longestCommandLength int

	deduplicator deduplicator

	OnUpdate chan struct{} // Call GetProcesses() to get the updated list
}

func NewTracker() *Tracker {
	tracker := &Tracker{}
	tracker.OnUpdate = make(chan struct{}, 1)
	tracker.deduplicator = deduplicator{}

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

	// Restore original commands for dying processes
	preserveDyingProcessCommands(procs, tracker.current)

	for _, proc := range procs {
		tracker.deduplicator.register(proc)
	}
	for _, proc := range procs {
		disambiguator := tracker.deduplicator.disambiguator(proc)
		proc.DeduplicationSuffix = ""
		if disambiguator != "" {
			proc.DeduplicationSuffix = "[" + disambiguator + "]"
		}
	}

	longestCommandLength := 0
	longestCommand := ""
	for _, p := range procs {
		if len(p.Command)+len(p.DeduplicationSuffix) > longestCommandLength {
			longestCommandLength = len(p.Command) + len(p.DeduplicationSuffix)
			longestCommand = p.Command + p.DeduplicationSuffix
		}
	}

	procsMap := make(map[int]*Process)
	for _, p := range procs {
		procsMap[p.Pid] = p
	}

	tracker.mutex.Lock()

	if longestCommandLength > tracker.longestCommandLength {
		tracker.longestCommandLength = longestCommandLength
		log.Debugf("New longest command is %d chars: %q", longestCommandLength, longestCommand)
	}

	if tracker.current != nil {
		// Keep track of dead children from previous frame
		preserveDeadChildren(tracker.current, procsMap)

		// Update launch counts tree
		tracker.launches = updateLaunches(tracker.launches, tracker.current, procsMap)

		trackDeaths(tracker.current, procsMap)
	}

	fillInNativities(procsMap)

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

// preserveDyingProcessCommands restores the original command names for processes
// that are dying (shown with parenthesized names like "(bash)").
func preserveDyingProcessCommands(current []*Process, previous map[int]*Process) {
	if previous == nil {
		return
	}

	for _, proc := range current {
		// Check if this is a dying process (parenthesized command)
		isParenthesized := strings.HasPrefix(proc.cmdline, "(") && strings.HasSuffix(proc.cmdline, ")")
		if proc.cmdline != "<defunct>" && proc.cmdline != "<exiting>" && !isParenthesized {
			continue
		}

		// Look up the previous state of this process
		oldProc, found := previous[proc.Pid]
		if !found {
			continue
		}

		// Verify it's the same process using SameAs (checks PID and start time)
		if !oldProc.SameAs(*proc) {
			continue
		}

		// Restore the original command information
		proc.cmdline = oldProc.cmdline
		proc.Command = oldProc.Command
		proc.lowercaseCommand = oldProc.lowercaseCommand
	}
}
