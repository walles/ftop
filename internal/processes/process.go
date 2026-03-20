package processes

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/util"
)

// We launch these all the time, pretend we are not so to not mess up the
// launched commands view.
var hiddenSelfChildCommands = map[string]struct{}{
	"iostat":  {},
	"netstat": {},
	"ps":      {},
}

// How old children count towards a process' nativity?
const NATIVITY_MAX_AGE = 60 * time.Second

// At least on macOS, ps' elapsed-time metric (etime) comes without decimals
const ETIME_PRECISION = time.Second

// Slowest I have seen on my system was around 900ms with race detector enabled
// and the system "sleeping".
const MAX_PS_DURATION = 1_000 * time.Millisecond

// Tolerance for matching same-PID processes across snapshots. If PIDs match and
// start times match within this tolerance, we consider it the same process.
const SAME_PROCESS_STARTTIME_TOLERANCE = ETIME_PRECISION + MAX_PS_DURATION

type Process struct {
	Pid      int
	ppid     int // The init process can have 0 here, meaning it has no parent
	children []*Process
	parent   *Process

	cmdline string // "git clone git@github.com:walles/ftop.git"
	Command string // "git"

	// "[2]", for disambiguating multiple processes with the same Command, or "" if the command is already unique
	DeduplicationSuffix string

	lowercaseCommand string // "git"

	startTime time.Time

	Username string

	RssKb         int
	memoryPercent *float64

	cpuPercent *float64
	CpuTime    *time.Duration

	// Count of children younger than NATIVITY_MAX_AGE
	Nativity int

	// Birth timestamps for all now-dead children, used for nativity calculation
	deadChildrenBirthTimes []time.Time
}

// Match + group: " 7708 1 00:21 501 0.1 0:00.08 0.0 /usr/bin/sleep"
var PS_LINE = regexp.MustCompile(
	" *([0-9]+) +([0-9]+) +([0-9]+) +([^ ]+) +([^ ]+) +([0-9.]+) +([-0-9.:]+) +([0-9.]+) +(.*)",
)

// Match + group: "1:02.03"
var CPU_DURATION_OSX = regexp.MustCompile(`^([0-9]+):([0-9][0-9]\.[0-9]+)$`)

// Match + group: "00:21" and malformed "00:-1"
var ELAPSED_DURATION_MINUTES = regexp.MustCompile(`^([0-9]+):(-?[0-9]+)$`)

// Match + group: "01:23:45"
var CPU_DURATION_LINUX = regexp.MustCompile(`^([0-9][0-9]):([0-9][0-9]):([0-9][0-9])$`)

// Match + group: "123-01:23:45"
var CPU_DURATION_LINUX_DAYS = regexp.MustCompile(`^([0-9]+)-([0-9][0-9]):([0-9][0-9]):([0-9][0-9])$`)

var uidToUsernameCache = map[int]string{}

// Command name and PID.
// Example return value:
//
//	bash(1234)
func (p *Process) String() string {
	return fmt.Sprintf("%s(%d)", p.Command, p.Pid)
}

func (p *Process) Parent() *Process {
	return p.parent
}

func (p *Process) Children() []*Process {
	return p.children
}

func (p *Process) StartTime() time.Time {
	return p.startTime
}

// Command line split into arguments, with path coalescing matching command
// parsing. Example return value:
//
//	["/usr/bin/git", "clone", "git@github.com:walles/ftop.git"]
func (p *Process) CommandLine() []string {
	return cmdlineToSlice(p.cmdline, exists)
}

func uidToUsername(uid int) string {
	if username, found := uidToUsernameCache[uid]; found {
		return username
	}

	uidString := strconv.FormatInt(int64(uid), 10)
	username := uidString // Fallback when lookup fails

	user, err := user.LookupId(uidString)
	if err == nil {
		username = user.Username
	}

	uidToUsernameCache[uid] = username
	return username
}

// Convert a CPU duration string returned by ps to a Duration
func parseDuration(durationString string) (time.Duration, error) {
	if match := CPU_DURATION_OSX.FindStringSubmatch(durationString); match != nil {
		minutes, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, fmt.Errorf("failed to parse minutes <%s> from duration string <%s>: %v", match[1], durationString, err)
		}

		seconds, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse seconds <%s> from duration string <%s>: %v", match[2], durationString, err)
		}

		totalSeconds := float64(minutes*60) + seconds
		return time.Duration(totalSeconds * float64(time.Second)), nil
	}

	if match := CPU_DURATION_LINUX.FindStringSubmatch(durationString); match != nil {
		hours, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, fmt.Errorf("failed to parse hours <%s> from duration string <%s>: %v", match[1], durationString, err)
		}

		minutes, err := strconv.Atoi(match[2])
		if err != nil {
			return 0, fmt.Errorf("failed to parse minutes <%s> from duration string <%s>: %v", match[2], durationString, err)
		}

		seconds, err := strconv.Atoi(match[3])
		if err != nil {
			return 0, fmt.Errorf("failed to parse seconds <%s> from duration string <%s>: %v", match[3], durationString, err)
		}

		totalSeconds := (hours * 3600) + (minutes * 60) + seconds
		return time.Duration(totalSeconds * int(time.Second)), nil
	}

	if match := CPU_DURATION_LINUX_DAYS.FindStringSubmatch(durationString); match != nil {
		days, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, fmt.Errorf("failed to parse days <%s> from duration string <%s>: %v", match[1], durationString, err)
		}

		hours, err := strconv.Atoi(match[2])
		if err != nil {
			return 0, fmt.Errorf("failed to parse hours <%s> from duration string <%s>: %v", match[2], durationString, err)
		}

		minutes, err := strconv.Atoi(match[3])
		if err != nil {
			return 0, fmt.Errorf("failed to parse minutes <%s> from duration string <%s>: %v", match[3], durationString, err)
		}

		seconds, err := strconv.Atoi(match[4])
		if err != nil {
			return 0, fmt.Errorf("failed to parse seconds <%s> from duration string <%s>: %v", match[4], durationString, err)
		}

		totalSeconds := (days * 86400) + (hours * 3600) + (minutes * 60) + seconds
		return time.Duration(totalSeconds * int(time.Second)), nil
	}

	return 0, fmt.Errorf("failed to parse duration string <%s>", durationString)
}

func parseElapsedDuration(durationString string) (time.Duration, error) {
	if match := ELAPSED_DURATION_MINUTES.FindStringSubmatch(durationString); match != nil {
		minutes, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, fmt.Errorf("failed to parse minutes <%s> from elapsed duration string <%s>: %v", match[1], durationString, err)
		}

		seconds, err := strconv.Atoi(match[2])
		if err != nil {
			return 0, fmt.Errorf("failed to parse seconds <%s> from elapsed duration string <%s>: %v", match[2], durationString, err)
		}

		totalSeconds := (minutes * 60) + seconds
		if totalSeconds < 0 {
			return 0, nil
		}

		return time.Duration(totalSeconds) * time.Second, nil
	}

	if match := CPU_DURATION_LINUX.FindStringSubmatch(durationString); match != nil {
		hours, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, fmt.Errorf("failed to parse hours <%s> from elapsed duration string <%s>: %v", match[1], durationString, err)
		}

		minutes, err := strconv.Atoi(match[2])
		if err != nil {
			return 0, fmt.Errorf("failed to parse minutes <%s> from elapsed duration string <%s>: %v", match[2], durationString, err)
		}

		seconds, err := strconv.Atoi(match[3])
		if err != nil {
			return 0, fmt.Errorf("failed to parse seconds <%s> from elapsed duration string <%s>: %v", match[3], durationString, err)
		}

		totalSeconds := (hours * 3600) + (minutes * 60) + seconds
		return time.Duration(totalSeconds) * time.Second, nil
	}

	if match := CPU_DURATION_LINUX_DAYS.FindStringSubmatch(durationString); match != nil {
		days, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, fmt.Errorf("failed to parse days <%s> from elapsed duration string <%s>: %v", match[1], durationString, err)
		}

		hours, err := strconv.Atoi(match[2])
		if err != nil {
			return 0, fmt.Errorf("failed to parse hours <%s> from elapsed duration string <%s>: %v", match[2], durationString, err)
		}

		minutes, err := strconv.Atoi(match[3])
		if err != nil {
			return 0, fmt.Errorf("failed to parse minutes <%s> from elapsed duration string <%s>: %v", match[3], durationString, err)
		}

		seconds, err := strconv.Atoi(match[4])
		if err != nil {
			return 0, fmt.Errorf("failed to parse seconds <%s> from elapsed duration string <%s>: %v", match[4], durationString, err)
		}

		totalSeconds := (days * 86400) + (hours * 3600) + (minutes * 60) + seconds
		return time.Duration(totalSeconds) * time.Second, nil
	}

	return 0, fmt.Errorf("failed to parse elapsed duration string <%s>", durationString)
}

func psLineToProcess(line string, snapshotTime time.Time) (*Process, error) {
	match := PS_LINE.FindStringSubmatch(line)
	if match == nil {
		return nil, fmt.Errorf("failed to match ps line <%q>", line)
	}

	pid, err := strconv.Atoi(match[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse pid <%s> from line <%s>: %v", match[1], line, err)
	}

	ppid, err := strconv.Atoi(match[2])
	if err != nil {
		return nil, fmt.Errorf("failed to parse ppid <%s> from line <%s>: %v", match[2], line, err)
	}

	rss_kb, err := strconv.Atoi(match[3])
	if err != nil {
		return nil, fmt.Errorf("failed to parse rss_kb <%s> from line <%s>: %v", match[3], line, err)
	}

	elapsedString := match[4]
	elapsed, err := parseElapsedDuration(elapsedString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse elapsed time <%s> from line <%s>: %v", match[4], line, err)
	}

	// startTime comes from ps wall-clock data, so strip any monotonic component
	// inherited from time.Now() to avoid monotonic-based Sub() deltas.
	startTime := snapshotTime.Round(0).Add(-elapsed)

	uid, err := strconv.Atoi(match[5])
	if err != nil {
		return nil, fmt.Errorf("failed to parse UID <%s> from line <%s>: %v", match[5], line, err)
	}
	username := uidToUsername(uid)

	cpu_percent, err := strconv.ParseFloat(match[6], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cpu_percent <%s> from line <%s>: %v", match[6], line, err)
	}

	cpu_time, err := parseDuration(match[7])
	if err != nil {
		return nil, fmt.Errorf("failed to parse cpu_time <%s> from line <%s>: %v", match[7], line, err)
	}

	memory_percent, err := strconv.ParseFloat(match[8], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory_percent <%s> from line <%s>: %v", match[8], line, err)
	}

	cmdline := match[9]
	command := cmdlineToCommand(cmdline)

	return &Process{
		Pid:              pid,
		ppid:             ppid,
		RssKb:            rss_kb,
		startTime:        startTime,
		Username:         username,
		cpuPercent:       &cpu_percent,
		CpuTime:          &cpu_time,
		memoryPercent:    &memory_percent,
		cmdline:          cmdline,
		Command:          command,
		lowercaseCommand: strings.ToLower(command),
	}, nil
}

func GetAll() ([]*Process, error) {
	command := []string{
		"/bin/ps",
		"-ax",
		"-o",
		"pid=,ppid=,rss=,etime=,uid=,pcpu=,time=,%mem=,command=",
	}

	var processes map[int]*Process
	attempt := 0
	for {
		attempt += 1
		processes = make(map[int]*Process, 0)

		startedAt := time.Now()
		err := util.Exec(command, func(line string) error {
			proc, err := psLineToProcess(line, startedAt)
			if err != nil {
				return err
			}

			processes[proc.Pid] = proc
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to get process list: %v", err)
		}

		duration := time.Since(startedAt)
		if duration <= MAX_PS_DURATION {
			break
		}

		// We can get here if the system has been hibernated and resumed,
		// causing the clock to jump forward while ps is running.

		if attempt >= 5 {
			return nil, fmt.Errorf(
				"ps command took too long (%s > %s) even at attempt %d",
				util.FormatDuration(duration),
				util.FormatDuration(MAX_PS_DURATION),
				attempt,
			)
		}

		log.Infof(
			"/bin/ps run took %s (> %s) wall clock time, retrying",
			util.FormatDuration(duration),
			util.FormatDuration(MAX_PS_DURATION),
		)
	}

	// Resolve parent-child relationships
	resolveLinks(processes)

	// Without this our every-second calls to ps will mess up the launched
	// commands view.
	removeSelfChildren(processes, os.Getpid())

	processList := make([]*Process, 0, len(processes))
	for _, proc := range processes {
		processList = append(processList, proc)
	}
	return processList, nil
}

// On entry, this function assumes that all processes have a "ppid" field
// containing the PID of their parent process.
//
// When done, all processes will have a "parent" field with a reference to the
// process' parent process object.
//
// Also, all processes will have a (possibly empty) "children" field containing
// a set of references to child processes.
func resolveLinks(processes map[int]*Process) {
	for _, proc := range processes {
		if proc.ppid == 0 {
			if proc.Pid != 1 {
				log.Infof("Non-init process without parent PID: %s", proc.String())
			}
			proc.parent = nil
			continue
		}

		parent, found := processes[proc.ppid]
		if !found {
			log.Infof("Failed to find parent process %d for process %s", proc.ppid, proc.String())
			proc.parent = nil
			continue
		}
		proc.parent = parent

		// Found our parent, say hello!
		proc.parent.children = append(proc.parent.children, proc)
	}
}

func removeSelfChildren(processes map[int]*Process, selfPid int) {
	selfProcess, found := processes[selfPid]
	if !found {
		return
	}

	keptChildren := make([]*Process, 0)
	toDelete := make([]int, 1)
	for _, child := range selfProcess.children {
		if !shouldHideSelfChild(child) {
			keptChildren = append(keptChildren, child)
			continue
		}

		child.parent = nil
		toDelete = append(toDelete, child.Pid)
	}
	for _, pid := range toDelete {
		delete(processes, pid)
	}

	selfProcess.children = keptChildren
}

func shouldHideSelfChild(child *Process) bool {
	_, shouldHide := hiddenSelfChildCommands[child.Command]
	return shouldHide
}

func fillInNativities(processes map[int]*Process) {
	now := time.Now()

	for _, proc := range processes {
		nativity := 0

		// Living children
		for _, child := range proc.children {
			age := now.Sub(child.startTime)
			if age > NATIVITY_MAX_AGE {
				continue
			}

			nativity += 1
		}

		// Dead children
		for _, birthTime := range proc.deadChildrenBirthTimes {
			age := now.Sub(birthTime)
			if age > NATIVITY_MAX_AGE {
				continue
			}

			nativity += 1
		}

		proc.Nativity = nativity
	}
}

func (p *Process) CpuPercentString() string {
	if p.cpuPercent == nil {
		return "--"
	}

	return fmt.Sprintf("%.0f%%", *p.cpuPercent)
}

func (p *Process) RamPercentString() string {
	if p.memoryPercent == nil {
		return "--"
	}

	return fmt.Sprintf("%.0f%%", *p.memoryPercent)
}

// Converts cpuTime to a string. Example outputs:
//
//	45s
//	39m02s
//	2h09m
//	1d04h
func (p *Process) CpuTimeString() string {
	if p.CpuTime == nil {
		return "--"
	}

	return util.FormatDuration(*p.CpuTime)
}

func (p *Process) CpuTimeOrZero() time.Duration {
	if p.CpuTime == nil {
		return 0
	}

	return *p.CpuTime
}

func (p *Process) SameAs(other *Process) bool {
	if p.Pid != other.Pid {
		return false
	}

	delta := p.startTime.Sub(other.startTime).Abs()

	return delta <= SAME_PROCESS_STARTTIME_TOLERANCE
}

func (p *Process) IsAlive() bool {
	err := syscall.Kill(p.Pid, 0)
	if err == nil {
		return true
	}

	return errors.Is(err, syscall.EPERM)
}
