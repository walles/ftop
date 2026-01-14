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
