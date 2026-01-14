package io

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/walles/ptop/internal/util"
)

// Matches output lines in "netstat -ib" on macOS.
//
// Extracted columns are interface name, incoming bytes count and outgoing bytes
// count.
//
// If you look carefully at the output, this regex will only match lines with
// error counts, which is only one line per interface.
var NETSTAT_BNI_LINE_RE = regexp.MustCompile(`^([^ ]+).*[0-9]+ +([0-9]+) +[0-9]+ +[0-9]+ +([0-9]+) +[0-9]+$`)

// NOTE: iostat does not give us separate values for read and write, so we can't
// tell them apart.

func GetNetworkStats() (map[string]uint64, error) {
	result := make(map[string]uint64)

	command := []string{"netstat", "-bni"}
	shouldSkipHeader := true
	err := util.Exec(command, func(line string) error {
		if shouldSkipHeader {
			shouldSkipHeader = false
			return nil
		}

		matches := NETSTAT_BNI_LINE_RE.FindStringSubmatch(line)
		if matches == nil {
			return nil // Ignore non-matching lines
		}

		ifName := matches[1]
		inBytes, err := strconv.ParseUint(matches[2], 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse incoming bytes count %q for netstat line <%s>: %v", matches[2], line, err)
		}

		outBytes, err := strconv.ParseUint(matches[3], 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse outgoing bytes count %q for netstat line <%s>: %v", matches[3], line, err)
		}

		if _, exists := result[ifName]; !exists {
			result[ifName] = 0
		}
		result[ifName] += inBytes + outBytes

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get network stats: %v", err)
	}

	return result, nil
}

func GetDiskStats() (map[string]uint64, error) {
	result := make(map[string]uint64)

	command := []string{"iostat", "-dKI", "-n 99"}

	lines := []string{}
	err := util.Exec(command, func(line string) error {
		if len(lines) >= 3 {
			return fmt.Errorf("expected exactly three lines but just got line four")
		}

		lines = append(lines, line)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get disk stats: %v", err)
	}

	if len(lines) != 3 {
		return nil, fmt.Errorf("expected 3 lines but got %d", len(lines))
	}

	// Example: ["disk0"]
	deviceNames := strings.Fields(lines[0])

	// Example: ["15.95", "2816998", "43889.27"]
	//
	// Numbers are: "kB/t", "xfrs" and "MB"
	numbers := strings.Fields(lines[2])
	if len(numbers) != len(deviceNames)*3 {
		return nil, fmt.Errorf(
			"expected %d numbers for %d device names but got %d\nnumbers: %v\ndevice names: %v",
			len(deviceNames)*3,
			len(deviceNames),
			len(numbers),
			numbers,
			deviceNames,
		)
	}

	for i, deviceName := range deviceNames {
		if _, exists := result[deviceName]; exists {
			return nil, fmt.Errorf("duplicate disk name %q found in %v", deviceName, deviceNames)
		}

		mbString := numbers[i*3+2]
		mbFloat, err := strconv.ParseFloat(mbString, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MB number %q for device %q: %v", mbString, deviceName, err)
		}

		bytes := uint64(mbFloat * 1024 * 1024)
		result[deviceName] = bytes
	}

	return result, nil
}
