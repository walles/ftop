package io

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/walles/ptop/internal/util"
)

// Matches output lines in "netstat -ib" on macOS.
//
// Extracted columns are interface name, incoming bytes count and outgoing bytes
// count.
//
// If you look carefully at the output, this regex will only match lines with
// error counts, which is only one line per interface.
var NETSTAT_IB_LINE_RE = regexp.MustCompile(`^([^ ]+).*[0-9]+ +([0-9]+) +[0-9]+ +[0-9]+ +([0-9]+) +[0-9]+$`)

func GetNetworkStats() (map[string]uint64, error) {
	result := make(map[string]uint64)

	command := []string{"netstat", "-bni"}
	shouldSkipHeader := true
	err := util.Exec(command, func(line string) error {
		if shouldSkipHeader {
			shouldSkipHeader = false
			return nil
		}

		matches := NETSTAT_IB_LINE_RE.FindStringSubmatch(line)
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
