package processes

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/util"
)

// How old children count towards a process' nativity?
const NATIVITY_MAX_AGE = 60 * time.Second

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
	Nativity uint

	// Birth timestamps for all now-dead children, used for nativity calculation
	// FIXME: Make sure we populate this field!
	deadChildrenBirthTimes []time.Time
}

// Match + group: " 7708 1 Mon Mar  7 09:33:11 2016  netbios 0.1 0:00.08  0.0 /usr/sbin/netbiosd hj"
var PS_LINE = regexp.MustCompile(
	" *([0-9]+) +([0-9]+) +([0-9]+) +([A-Za-z0-9: ]+) +([^ ]+) +([0-9.]+) +([-0-9.:]+) +([0-9.]+) +(.*)",
)

// Match + group: "1:02.03"
var CPU_DURATION_OSX = regexp.MustCompile(`^([0-9]+):([0-9][0-9]\.[0-9]+)$`)

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

func (p *Process) StartTime() time.Time {
	return p.startTime
}

// Parse a local date from ps into a datetime.datetime object.
//
// Example inputs:
//
//	Wed Dec 16 12:41:43 2020
//	Sat Jan  9 14:20:34 2021
func parseTime(time_string string) (time.Time, error) {
	monthLetters := time_string[4:7]
	monthNames := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	monthIndex := slices.Index(monthNames, monthLetters) // Zero based
	if monthIndex == -1 {
		return time.Time{}, fmt.Errorf("failed to parse month <%s> from time string <%s>", monthLetters, time_string)
	}

	dayOfMonth, err := strconv.Atoi(strings.TrimSpace(time_string[8:10]))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse day of month <%s> from time string <%s>: %v", time_string[8:10], time_string, err)
	}

	hour, err := strconv.Atoi(strings.TrimSpace(time_string[11:13]))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse hour <%s> from time string <%s>: %v", time_string[11:13], time_string, err)
	}

	minute, err := strconv.Atoi(strings.TrimSpace(time_string[14:16]))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse minute <%s> from time string <%s>: %v", time_string[14:16], time_string, err)
	}

	second, err := strconv.Atoi(strings.TrimSpace(time_string[17:19]))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse second <%s> from time string <%s>: %v", time_string[17:19], time_string, err)
	}

	year, err := strconv.Atoi(strings.TrimSpace(time_string[20:24]))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse year <%s> from time string <%s>: %v", time_string[20:24], time_string, err)
	}

	return time.Date(year, time.Month(monthIndex+1), dayOfMonth, hour, minute, second, 0, time.Local), nil
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

func psLineToProcess(line string) (*Process, error) {
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

	start_time_string := match[4]
	start_time, err := parseTime(start_time_string)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start_time <%s> from line <%s>: %v", match[4], line, err)
	}

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
		startTime:        start_time,
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
	processes := make(map[int]*Process, 0)

	command := []string{
		"/bin/ps",
		"-ax",
		"-o",
		"pid=,ppid=,rss=,lstart=,uid=,pcpu=,time=,%mem=,command=",
	}
	err := util.Exec(command, func(line string) error {
		proc, err := psLineToProcess(line)
		if err != nil {
			return err
		}

		processes[proc.Pid] = proc
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get process list: %v", err)
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

	// Remove all children from selfProcess
	toDelete := make([]int, 0)
	for _, child := range selfProcess.children {
		child.parent = nil
		toDelete = append(toDelete, child.Pid)
	}
	for _, pid := range toDelete {
		delete(processes, pid)
	}

	selfProcess.children = []*Process{}
}

func fillInNativities(processes map[int]*Process) {
	now := time.Now()

	for _, proc := range processes {
		var nativity uint = 0

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

func (p *Process) SameAs(other Process) bool {
	return p.Pid == other.Pid && p.startTime.Equal(other.startTime)
}

func (p *Process) IsAlive() bool {
	err := syscall.Kill(p.Pid, 0)
	if err == nil {
		return true
	}

	return errors.Is(err, syscall.EPERM)
}
