package processes

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

type Process struct {
	pid      int
	ppid     *int
	children []*Process
	parent   *Process

	cmdline           string // "git clone git@github.com:walles/px.git"
	command           string // "git"
	lowercase_command string // "git"

	start_time time.Time

	username string

	rss_kb         int
	memory_percent *float64

	cpu_percent         *float64
	cpu_time            *time.Duration
	aggregated_cpu_time time.Duration
}

// Match + group: " 7708 1 Mon Mar  7 09:33:11 2016  netbios 0.1 0:00.08  0.0 /usr/sbin/netbiosd hj"
var PS_LINE = regexp.MustCompile(
	" *([0-9]+) +([0-9]+) +([0-9]+) +([A-Za-z0-9: ]+) +([^ ]+) +([0-9.]+) +([-0-9.:]+) +([0-9.]+) +(.*)",
)

func (p *Process) String() string {
	return fmt.Sprintf("%s(%d)", p.command, p.pid)
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
	start_time, err := parse_time(start_time_string)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse start_time <%s> from line <%s>: %v", match[4], line, err)
	}

	uid, err := strconv.Atoi(match[5])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse UID <%s> from line <%s>: %v", match[5], line, err)
	}
	username := uid_to_username(uid)

	cpu_percent, err := strconv.ParseFloat(match[6], 64)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse cpu_percent <%s> from line <%s>: %v", match[6], line, err)
	}

	cpu_time, err := parse_duration(match[7])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse cpu_time <%s> from line <%s>: %v", match[7], line, err)
	}

	memory_percent, err := strconv.ParseFloat(match[8], 64)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse memory_percent <%s> from line <%s>: %v", match[8], line, err)
	}

	cmdline := match[9]

	return &Process{
		pid:            pid,
		ppid:           &ppid,
		rss_kb:         rss_kb,
		start_time:     start_time,
		username:       username,
		cpu_percent:    &cpu_percent,
		cpu_time:       &cpu_time,
		memory_percent: &memory_percent,
		cmdline:        cmdline,
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

	// FIXME: Spawn ps, then iterate the
}
