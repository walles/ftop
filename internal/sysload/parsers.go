package sysload

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// Parses /proc/cpuinfo and returns logical and physical core counts
func parseProcCpuInfo(procCpuInfoStr string) (coresLogical int, coresPhysical int, err error) {
	// Note the trailing spaces, they must be there for number extraction to work!
	const PROCESSOR_COUNT_PREFIX = "processor\t: "
	const CORE_ID_PREFIX = "core id\t\t: "

	coreIds := map[int]struct{}{}
	maxProcessorNo := 0

	scanner := bufio.NewScanner(strings.NewReader(procCpuInfoStr))
	for scanner.Scan() {
		line := scanner.Text()
		if processorNoString, ok := strings.CutPrefix(line, PROCESSOR_COUNT_PREFIX); ok {
			processorNo, err := strconv.Atoi(processorNoString)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to parse processor number: %w", err)
			}

			if processorNo > maxProcessorNo {
				maxProcessorNo = processorNo
			}
		} else if coreIdString, ok := strings.CutPrefix(line, CORE_ID_PREFIX); ok {
			coreId, err := strconv.Atoi(coreIdString)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to parse core id: %w", err)
			}
			coreIds[coreId] = struct{}{}
		}
	}

	physical := len(coreIds)
	logical := maxProcessorNo + 1 // Processor numbers are zero-based

	if physical == 0 {
		// I get this on my cell phone
		physical = logical
	}

	return logical, physical, nil
}

func parseProcMemInfo(procMemInfoStr string) (usedBytes uint64, totalBytes uint64, err error) {
	var (
		totalKB, availableKB, freeKB, buffersKB, cachedKB *uint64
		swapcachedKB, swaptotalKB, swapfreeKB             *uint64
	)

	scanner := bufio.NewScanner(strings.NewReader(procMemInfoStr))
	for scanner.Scan() {
		line := scanner.Text()

		// helper to parse a kB value from lines like "MemTotal:       16384256 kB"
		parseIf := func(name string) (*uint64, bool) {
			if !strings.HasPrefix(line, name+":") {
				return nil, false
			}

			parts := strings.Fields(line)
			if len(parts) < 2 {
				return nil, false
			}

			v, err := strconv.ParseUint(parts[1], 10, 64)
			if err != nil {
				return nil, false
			}

			return &v, true
		}

		if v, ok := parseIf("MemTotal"); ok && totalKB == nil {
			totalKB = v
			continue
		}

		if v, ok := parseIf("MemAvailable"); ok && availableKB == nil {
			availableKB = v
			continue
		}

		if v, ok := parseIf("MemFree"); ok && freeKB == nil {
			freeKB = v
			continue
		}

		if v, ok := parseIf("Buffers"); ok && buffersKB == nil {
			buffersKB = v
			continue
		}

		if v, ok := parseIf("Cached"); ok && cachedKB == nil {
			cachedKB = v
			continue
		}

		if v, ok := parseIf("SwapCached"); ok && swapcachedKB == nil {
			swapcachedKB = v
			continue
		}

		if v, ok := parseIf("SwapTotal"); ok && swaptotalKB == nil {
			swaptotalKB = v
			continue
		}

		if v, ok := parseIf("SwapFree"); ok && swapfreeKB == nil {
			swapfreeKB = v
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("failed scanning meminfo: %w", err)
	}

	if totalKB == nil {
		return 0, 0, fmt.Errorf("MemTotal not found in meminfo")
	}
	if swaptotalKB == nil || swapfreeKB == nil {
		return 0, 0, fmt.Errorf("SwapTotal or SwapFree not found in meminfo")
	}

	swapUsedKB := *swaptotalKB - *swapfreeKB

	if availableKB != nil {
		// Prefer MemAvailable if present
		ramUsedKB := *totalKB - *availableKB

		totalBytes = *totalKB * 1024
		usedBytes = (swapUsedKB + ramUsedKB) * 1024

		return usedBytes, totalBytes, nil
	}

	if freeKB == nil || buffersKB == nil || cachedKB == nil || swapcachedKB == nil {
		return 0, 0, fmt.Errorf("insufficient fields in meminfo to calculate memory used")
	}

	ramUsedKB := *totalKB - (*freeKB + *buffersKB + *cachedKB + *swapcachedKB)

	totalBytes = *totalKB * 1024
	usedBytes = (swapUsedKB + ramUsedKB) * 1024

	return usedBytes, totalBytes, nil
}

func parseProcLoadAvg(procLoadAvgStr string) (load1M float64, load5M float64, load15M float64, err error) {
	parts := strings.Fields(procLoadAvgStr)
	if len(parts) < 3 {
		return 0, 0, 0, fmt.Errorf("unexpected format of /proc/loadavg")
	}

	load1M, err = strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse 1-minute load average: %w", err)
	}

	load5M, err = strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse 5-minute load average: %w", err)
	}

	load15M, err = strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse 15-minute load average: %w", err)
	}

	return load1M, load5M, load15M, nil
}
