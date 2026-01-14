package io

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func parseProcNetDev(procNetDevStr string) (map[string]uint64, error) {
	// Parse one line of /proc/net/dev
	//
	// Example input (includes leading whitespace):
	//   eth0: 29819439   19890    0    0    0     0          0         0   364327    6584    0    0    0     0       0          0
	lineRegexp := regexp.MustCompile(`^ *([^:]+): +([0-9]+) +[0-9]+ +[0-9]+ +[0-9]+ +[0-9]+ +[0-9]+ +[0-9]+ +[0-9]+ +([0-9]+)[0-9 ]+$`)

	result := make(map[string]uint64)

	scanner := bufio.NewScanner(strings.NewReader(procNetDevStr))
	for scanner.Scan() {
		line := scanner.Text()
		match := lineRegexp.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		iface := strings.TrimSpace(match[1])
		inBytesStr := match[2]
		outBytesStr := match[3]

		inBytes, err := strconv.ParseUint(inBytesStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing input byte counts failed: %w", err)
		}

		outBytes, err := strconv.ParseUint(outBytesStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing output byte counts failed: %w", err)
		}

		if inBytes == 0 && outBytes == 0 {
			continue
		}

		result[iface+" (in)"] = inBytes
		result[iface+" (out)"] = outBytes
	}

	return result, nil
}

func parseProcDiskstats(procDiskstatsStr string) (map[string]uint64, error) {
	// Parse a line from /proc/diskstats
	//
	// First group is the name. To get partitions rather than disks we require
	// the name to end in a number.
	//
	// Second and third groups are sector reads and writes respectively.
	// convert to bytes.
	//
	// Line format documented here:
	// https://www.kernel.org/doc/Documentation/admin-guide/iostats.rst
	lineRegexp := regexp.MustCompile(`^ *[0-9]+ +[0-9]+ +([a-z]+[0-9]+) +[0-9]+ +[0-9]+ +([0-9]+) +[0-9]+ +[0-9]+ +[0-9]+ +([0-9]+) .*$`)

	result := make(map[string]uint64)

	scanner := bufio.NewScanner(strings.NewReader(procDiskstatsStr))
	for scanner.Scan() {
		line := scanner.Text()
		match := lineRegexp.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		name := match[1]
		readSectorsStr := match[2]
		writeSectorsStr := match[3]

		readSectors, err := strconv.ParseUint(readSectorsStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing read sectors failed: %w", err)
		}

		writeSectors, err := strconv.ParseUint(writeSectorsStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing write sectors failed: %w", err)
		}

		if readSectors == 0 && writeSectors == 0 {
			continue
		}

		// Multiply by 512 to get bytes from sectors:
		// https://stackoverflow.com/a/38136179/473672
		result[name+" (read)"] = readSectors * 512
		result[name+" (write)"] = writeSectors * 512
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning input failed: %w", err)
	}

	return result, nil
}
