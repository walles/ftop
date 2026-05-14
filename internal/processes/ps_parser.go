package processes

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/util"
)

// Convert a CPU duration string returned by ps to a Duration
func parseDuration(s string) (time.Duration, error) {
	parts := strings.SplitN(s, ":", 3)

	if len(parts) == 2 {
		// Example: "1:02.03"
		minutes, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("failed to parse minutes <%s> from duration string <%s>: %v", parts[0], s, err)
		}

		seconds, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse seconds <%s> from duration string <%s>: %v", parts[1], s, err)
		}

		totalSeconds := float64(minutes*60) + seconds
		return time.Duration(totalSeconds * float64(time.Second)), nil
	}

	if len(parts) == 3 {
		// Examples: "01:23:45", "123-01:23:45"
		days := 0
		hoursStr := parts[0]
		if dashIdx := strings.IndexByte(hoursStr, '-'); dashIdx > 0 {
			d, err := strconv.Atoi(hoursStr[:dashIdx])
			if err != nil {
				return 0, fmt.Errorf("failed to parse days from duration string <%s>: %v", s, err)
			}
			days = d
			hoursStr = hoursStr[dashIdx+1:]
		}

		hours, err := strconv.Atoi(hoursStr)
		if err != nil {
			return 0, fmt.Errorf("failed to parse hours <%s> from duration string <%s>: %v", hoursStr, s, err)
		}

		minutes, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("failed to parse minutes <%s> from duration string <%s>: %v", parts[1], s, err)
		}

		seconds, err := strconv.Atoi(parts[2])
		if err != nil {
			return 0, fmt.Errorf("failed to parse seconds <%s> from duration string <%s>: %v", parts[2], s, err)
		}

		totalSeconds := (days * 86400) + (hours * 3600) + (minutes * 60) + seconds
		return time.Duration(totalSeconds) * time.Second, nil
	}

	return 0, fmt.Errorf("failed to parse duration string <%s>", s)
}

func parseElapsedDuration(s string) (time.Duration, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) == 2 {
		// Examples: "00:21", malformed "00:-1" and malformed "-14:-1"
		minutes, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("failed to parse minutes <%s> from elapsed duration string <%s>: %v", parts[0], s, err)
		}
		seconds, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("failed to parse seconds <%s> from elapsed duration string <%s>: %v", parts[1], s, err)
		}
		return time.Duration((minutes*60)+seconds) * time.Second, nil
	}
	if len(parts) == 3 {
		// Examples: "01:23:45", "123-01:23:45"
		days := 0
		hoursStr := parts[0]
		if dashIdx := strings.IndexByte(hoursStr, '-'); dashIdx > 0 {
			d, err := strconv.Atoi(hoursStr[:dashIdx])
			if err != nil {
				return 0, fmt.Errorf("failed to parse days from elapsed duration string <%s>: %v", s, err)
			}
			days = d
			hoursStr = hoursStr[dashIdx+1:]
		}
		hours, err := strconv.Atoi(hoursStr)
		if err != nil {
			return 0, fmt.Errorf("failed to parse hours <%s> from elapsed duration string <%s>: %v", hoursStr, s, err)
		}
		minutes, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("failed to parse minutes <%s> from elapsed duration string <%s>: %v", parts[1], s, err)
		}
		seconds, err := strconv.Atoi(parts[2])
		if err != nil {
			return 0, fmt.Errorf("failed to parse seconds <%s> from elapsed duration string <%s>: %v", parts[2], s, err)
		}
		totalSeconds := (days * 86400) + (hours * 3600) + (minutes * 60) + seconds
		return time.Duration(totalSeconds) * time.Second, nil
	}
	return 0, fmt.Errorf("failed to parse elapsed duration string <%s>", s)
}

func processFieldsToProcess(fields [9]string, line string, snapshotTime time.Time) (*Process, error) {
	if fields[8] == "" {
		return nil, fmt.Errorf("failed to match ps line <%q>", line)
	}

	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse pid <%s> from line <%s>: %v", fields[0], line, err)
	}

	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse ppid <%s> from line <%s>: %v", fields[1], line, err)
	}

	rss_kb, err := strconv.Atoi(fields[2])
	if err != nil {
		return nil, fmt.Errorf("failed to parse rss_kb <%s> from line <%s>: %v", fields[2], line, err)
	}

	elapsedString := fields[3]
	elapsed, err := parseElapsedDuration(elapsedString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse elapsed time <%s> from line <%s>: %v", fields[3], line, err)
	}

	if elapsed < 0 {
		log.Debugf("/bin/ps reported process elapsed time <%s> in the future by %s from line <%s>",
			elapsedString, util.FormatDuration(elapsed.Abs()), line)
	}

	// startTime comes from ps wall-clock data, so strip any monotonic component
	// inherited from time.Now() to avoid monotonic-based Sub() deltas.
	startTime := snapshotTime.Round(0).Add(-elapsed)

	uid, err := strconv.Atoi(fields[4])
	if err != nil {
		return nil, fmt.Errorf("failed to parse UID <%s> from line <%s>: %v", fields[4], line, err)
	}
	username := uidToUsername(uid)

	cpu_percent, err := strconv.ParseFloat(fields[5], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cpu_percent <%s> from line <%s>: %v", fields[5], line, err)
	}

	cpu_time, err := parseDuration(fields[6])
	if err != nil {
		return nil, fmt.Errorf("failed to parse cpu_time <%s> from line <%s>: %v", fields[6], line, err)
	}

	memory_percent, err := strconv.ParseFloat(fields[7], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory_percent <%s> from line <%s>: %v", fields[7], line, err)
	}

	cmdline := fields[8]

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

func psLineToProcess(line string, snapshotTime time.Time) (*Process, error) {
	var fields [9]string
	fieldIdx := 0
	start := 0
	inWord := false

	for i := 0; i < len(line); i++ {
		if line[i] != ' ' {
			if !inWord {
				inWord = true
				start = i
			}
		} else {
			if inWord {
				fields[fieldIdx] = line[start:i]
				fieldIdx++
				inWord = false
				if fieldIdx == 8 {
					// The 9th field is the command string, which may contain spaces.
					// Find the start of the 9th field and grab the rest of the string.
					remaining := line[i:]
					trimStart := 0
					for trimStart < len(remaining) && remaining[trimStart] == ' ' {
						trimStart++
					}
					fields[8] = remaining[trimStart:]
					break
				}
			}
		}
	}

	return processFieldsToProcess(fields, line, snapshotTime)
}
