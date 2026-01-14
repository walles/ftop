package sysload

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func getMemoryUsage() (usedBytes uint64, totalBytes uint64, err error) {
	FIXME
}

func getCpuCoreCounts() (coresLogical int, coresPhysical int, err error) {
	// Note the trailing spaces, they must be there for number extraction to work!
	const PROCESSOR_COUNT_PREFIX = "processor\t: "
	const CORE_ID_PREFIX = "core id\t\t: "

	coreIds := map[int]struct{}{}
	maxProcessorNo := 0

	// Iterate over lines in /proc/cpuinfo
	cpuInfo, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return 0, 0, err
	}

	scanner := bufio.NewScanner(cpuInfo)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, PROCESSOR_COUNT_PREFIX) {
			processorNoString := strings.TrimPrefix(line, PROCESSOR_COUNT_PREFIX)
			processorNo, err := strconv.Atoi(processorNoString)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to parse processor number: %w", err)
			}

			if processorNo > maxProcessorNo {
				maxProcessorNo = processorNo
			}
		} else if strings.HasPrefix(line, CORE_ID_PREFIX) {
			coreIdString := strings.TrimPrefix(line, CORE_ID_PREFIX)
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

func getLoadAverages() (load1M float64, load5M float64, load15M float64, err error) {
	FIXME
}
