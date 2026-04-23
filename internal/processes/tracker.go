package processes

import (
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/util"
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

		interval := 1 * time.Second
		for {
			t0 := time.Now()
			time.Sleep(interval)
			sleepDuration := time.Since(t0)
			if sleepDuration > interval*2 {
				// This can happen on mac laptops when they go to sleep. And in
				// that situation, ps (as run by tracker.update() below) can
				// take a very long time. Which will mess up process start time
				// calculations. And when start times are off, that will mess up
				// PID reuse detection, leading to us claiming a lot of new
				// processes recently showed up, even though that never
				// happened.
				//
				// So let's just wait another interval to save us some trouble.
				log.Debugf("Sleeping %s took %s, retrying...", util.FormatDuration(interval), util.FormatDuration(sleepDuration))
				continue
			}

			tracker.update()
		}
	}()

	return tracker
}

func isFatal(err error) bool {
	_, isTimeAnomaly := err.(timeAnomalyError)
	return !isTimeAnomaly
}

func (tracker *Tracker) update() {
	procs, err := GetAll()
	if err != nil {
		if isFatal(err) {
			log.Errorf("Process tracker refresh failed, keeping previous snapshot: %v", err)
		} else {
			log.Debugf("Process tracker refresh had a time anomaly, keeping previous snapshot: %v", err)
		}
		return
	}

	procsMap := make(map[int]*Process)
	for _, p := range procs {
		procsMap[p.Pid] = p
	}

	var matches ProcessMatching
	if tracker.current != nil {
		matches, err = buildProcessMatches(tracker.current, procsMap)
		if err != nil {
			if isFatal(err) {
				log.Errorf("Process tracker refresh failed, keeping previous snapshot: %v", err)
			} else {
				log.Debugf("Process tracker refresh had a time anomaly, keeping previous snapshot: %v", err)
			}
			return
		}
	}

	// Restore original commands for dying processes
	preserveDyingProcessCommands(matches)

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
		command := p.Command()
		if len(command)+len(p.DeduplicationSuffix) > longestCommandLength {
			longestCommandLength = len(command) + len(p.DeduplicationSuffix)
			longestCommand = command + p.DeduplicationSuffix
		}
	}

	tracker.mutex.Lock()

	if longestCommandLength > tracker.longestCommandLength {
		tracker.longestCommandLength = longestCommandLength
		log.Debugf("New longest command is %d chars: %q", longestCommandLength, longestCommand)
	}

	if tracker.current != nil {
		// Keep track of dead children from previous frame
		preserveDeadChildren(matches)

		// Update launch counts tree
		tracker.launches = updateLaunches(tracker.launches, matches)

		trackDeaths(matches)
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

			// SameAs() protects against reused PIDs while tolerating etime's
			// one-second start time precision.
			if ok && proc.SameAs(baseProc) && proc.CpuTime != nil && baseProc.CpuTime != nil {
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
func preserveDyingProcessCommands(matching ProcessMatching) {
	if matching.Matched == nil {
		return
	}

	for _, proc := range matching.CurrentByPid {
		// Check if this is a dying process (parenthesized command)
		isParenthesized := strings.HasPrefix(proc.Cmdline, "(") && strings.HasSuffix(proc.Cmdline, ")")
		if proc.Cmdline != "<defunct>" && proc.Cmdline != "<exiting>" && !isParenthesized {
			continue
		}

		match, found := matching.Matched[proc.Pid]
		if !found {
			continue
		}

		// Restore the original command information
		proc.Cmdline = match.Old.Cmdline
	}
}
