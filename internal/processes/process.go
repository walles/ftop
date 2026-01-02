package processes

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/walles/ptop/internal/ui"
)

type Process struct {
	Pid      int
	ppid     *int
	children []*Process
	parent   *Process

	cmdline          string // "git clone git@github.com:walles/px.git"
	Command          string // "git"
	lowercaseCommand string // "git"

	startTime time.Time

	Username string

	RssKb         int
	memoryPercent *float64

	cpuPercent        *float64
	CpuTime           *time.Duration
	aggregatedCpuTime time.Duration
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

func (p *Process) String() string {
	return fmt.Sprintf("%s(%d)", p.Command, p.Pid)
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
		return time.Time{}, fmt.Errorf("Failed to parse month <%s> from time string <%s>", monthLetters, time_string)
	}

	dayOfMonth, err := strconv.Atoi(strings.TrimSpace(time_string[8:10]))
	if err != nil {
		return time.Time{}, fmt.Errorf("Failed to parse day of month <%s> from time string <%s>: %v", time_string[8:10], time_string, err)
	}

	hour, err := strconv.Atoi(strings.TrimSpace(time_string[11:13]))
	if err != nil {
		return time.Time{}, fmt.Errorf("Failed to parse hour <%s> from time string <%s>: %v", time_string[11:13], time_string, err)
	}

	minute, err := strconv.Atoi(strings.TrimSpace(time_string[14:16]))
	if err != nil {
		return time.Time{}, fmt.Errorf("Failed to parse minute <%s> from time string <%s>: %v", time_string[14:16], time_string, err)
	}

	second, err := strconv.Atoi(strings.TrimSpace(time_string[17:19]))
	if err != nil {
		return time.Time{}, fmt.Errorf("Failed to parse second <%s> from time string <%s>: %v", time_string[17:19], time_string, err)
	}

	year, err := strconv.Atoi(strings.TrimSpace(time_string[20:24]))
	if err != nil {
		return time.Time{}, fmt.Errorf("Failed to parse year <%s> from time string <%s>: %v", time_string[20:24], time_string, err)
	}

	return time.Date(year, time.Month(monthIndex+1), dayOfMonth, hour, minute, second, 0, time.Local), nil
}

func uidToUsername(uid int) string {
	if userName, found := uidToUsernameCache[uid]; found {
		return userName
	}

	uidString := strconv.FormatInt(int64(uid), 10)
	userName := uidString // Fallback when lookup fails

	user, err := user.LookupId(uidString)
	if err == nil {
		userName = user.Username
	}

	uidToUsernameCache[uid] = userName
	return userName
}

// Convert a CPU duration string returned by ps to a Duration
func parseDuration(durationString string) (time.Duration, error) {
	if match := CPU_DURATION_OSX.FindStringSubmatch(durationString); match != nil {
		minutes, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, fmt.Errorf("Failed to parse minutes <%s> from duration string <%s>: %v", match[1], durationString, err)
		}

		seconds, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			return 0, fmt.Errorf("Failed to parse seconds <%s> from duration string <%s>: %v", match[2], durationString, err)
		}

		totalSeconds := float64(minutes*60) + seconds
		return time.Duration(totalSeconds * float64(time.Second)), nil
	}

	if match := CPU_DURATION_LINUX.FindStringSubmatch(durationString); match != nil {
		hours, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, fmt.Errorf("Failed to parse hours <%s> from duration string <%s>: %v", match[1], durationString, err)
		}

		minutes, err := strconv.Atoi(match[2])
		if err != nil {
			return 0, fmt.Errorf("Failed to parse minutes <%s> from duration string <%s>: %v", match[2], durationString, err)
		}

		seconds, err := strconv.Atoi(match[3])
		if err != nil {
			return 0, fmt.Errorf("Failed to parse seconds <%s> from duration string <%s>: %v", match[3], durationString, err)
		}

		totalSeconds := (hours * 3600) + (minutes * 60) + seconds
		return time.Duration(totalSeconds * int(time.Second)), nil
	}

	if match := CPU_DURATION_LINUX_DAYS.FindStringSubmatch(durationString); match != nil {
		days, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, fmt.Errorf("Failed to parse days <%s> from duration string <%s>: %v", match[1], durationString, err)
		}

		hours, err := strconv.Atoi(match[2])
		if err != nil {
			return 0, fmt.Errorf("Failed to parse hours <%s> from duration string <%s>: %v", match[2], durationString, err)
		}

		minutes, err := strconv.Atoi(match[3])
		if err != nil {
			return 0, fmt.Errorf("Failed to parse minutes <%s> from duration string <%s>: %v", match[3], durationString, err)
		}

		seconds, err := strconv.Atoi(match[4])
		if err != nil {
			return 0, fmt.Errorf("Failed to parse seconds <%s> from duration string <%s>: %v", match[4], durationString, err)
		}

		totalSeconds := (days * 86400) + (hours * 3600) + (minutes * 60) + seconds
		return time.Duration(totalSeconds * int(time.Second)), nil
	}

	return 0, fmt.Errorf("Failed to parse duration string <%s>", durationString)
}

func psLineToProcess(line string) (*Process, error) {
	match := PS_LINE.FindStringSubmatch(line)
	if match == nil {
		return nil, fmt.Errorf("Failed to match ps line <%q>", line)
	}

	pid, err := strconv.Atoi(match[1])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse pid <%s> from line <%s>: %v", match[1], line, err)
	}

	ppid, err := strconv.Atoi(match[2])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse ppid <%s> from line <%s>: %v", match[2], line, err)
	}

	rss_kb, err := strconv.Atoi(match[3])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse rss_kb <%s> from line <%s>: %v", match[3], line, err)
	}

	start_time_string := match[4]
	start_time, err := parseTime(start_time_string)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse start_time <%s> from line <%s>: %v", match[4], line, err)
	}

	uid, err := strconv.Atoi(match[5])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse UID <%s> from line <%s>: %v", match[5], line, err)
	}
	username := uidToUsername(uid)

	cpu_percent, err := strconv.ParseFloat(match[6], 64)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse cpu_percent <%s> from line <%s>: %v", match[6], line, err)
	}

	cpu_time, err := parseDuration(match[7])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse cpu_time <%s> from line <%s>: %v", match[7], line, err)
	}

	memory_percent, err := strconv.ParseFloat(match[8], 64)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse memory_percent <%s> from line <%s>: %v", match[8], line, err)
	}

	cmdline := match[9]
	command := cmdlineToCommand(cmdline)

	return &Process{
		Pid:              pid,
		ppid:             &ppid,
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
	processes := []*Process{}

	command := []string{
		"/bin/ps",
		"-ax",
		"-o",
		"pid=,ppid=,rss=,lstart=,uid=,pcpu=,time=,%mem=,command=",
	}
	cmd := exec.Command(command[0], command[1:]...)

	env := []string{}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "LANG") || strings.HasPrefix(e, "LC_") {
			continue
		}
		env = append(env, e)
	}
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("Failed to get stdout pipe for ps: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("Failed to start ps: %v", err)
	}

	scanner := bufio.NewScanner(stdout)
	var readErr error
	for scanner.Scan() {
		line := scanner.Text()

		proc, err := psLineToProcess(line)
		if err != nil {
			if readErr == nil {
				readErr = fmt.Errorf("Failed to parse ps line: %v", err)
			}
			continue
		}

		processes = append(processes, proc)
	}

	if err := scanner.Err(); err != nil {
		if readErr == nil {
			readErr = fmt.Errorf("Error reading ps output: %v", err)
		}
	}

	if err := cmd.Wait(); err != nil {
		if readErr == nil {
			readErr = fmt.Errorf("ps command failed: %v", err)
		}
	}

	if readErr != nil {
		return nil, readErr
	}

	// FIXME: Resolve parent-child relationships

	// FIXME: Censor out ourselves

	return processes, nil
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

	return ui.FormatDuration(*p.CpuTime)
}
