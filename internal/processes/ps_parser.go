package processes

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/util"
)

type psParser struct {
}

func newPsParser() *psParser {
	return &psParser{}
}

// Match + group: " 7708 1 00:21 501 0.1 0:00.08 0.0 /usr/bin/sleep"
var PS_LINE = regexp.MustCompile(
	" *([0-9]+) +([0-9]+) +([0-9]+) +([^ ]+) +([^ ]+) +([0-9.]+) +([-0-9.:]+) +([0-9.]+) +(.*)",
)

// Match + group: "1:02.03"
var CPU_DURATION_OSX = regexp.MustCompile(`^([0-9]+):([0-9][0-9]\.[0-9]+)$`)

// Match + group: "00:21", malformed "00:-1" and malformed "-14:-1"
var ELAPSED_DURATION_MINUTES = regexp.MustCompile(`^(-?[0-9]+):(-?[0-9]+)$`)

// Match + group: "01:23:45"
var CPU_DURATION_LINUX = regexp.MustCompile(`^([0-9][0-9]):([0-9][0-9]):([0-9][0-9])$`)

// Match + group: "123-01:23:45"
var CPU_DURATION_LINUX_DAYS = regexp.MustCompile(`^([0-9]+)-([0-9][0-9]):([0-9][0-9]):([0-9][0-9])$`)

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

	if elapsed < 0 {
		log.Debugf("/bin/ps reported process elapsed time <%s> in the future by %s from line <%s>",
			elapsedString, util.FormatDuration(elapsed.Abs()), line)
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

	return &Process{
		Pid:           pid,
		ppid:          ppid,
		RssKb:         rss_kb,
		startTime:     startTime,
		Username:      username,
		cpuPercent:    &cpu_percent,
		CpuTime:       &cpu_time,
		memoryPercent: &memory_percent,
		Cmdline:       cmdline,
	}, nil
}

func (p *psParser) ParseLine(line string, snapshotTime time.Time) (*Process, error) {
	// For now, this just calls the existing regex-based parsing function
	// to establish our baseline performance.
	return psLineToProcess(line, snapshotTime)
}
