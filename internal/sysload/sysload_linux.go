package sysload

import (
	"os"
)

func getMemoryUsage() (usedBytes uint64, totalBytes uint64, err error) {
	FIXME
}

func getCpuCoreCounts() (coresLogical int, coresPhysical int, err error) {
	cpuInfo, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 0, 0, err
	}

	return parseProcCpuInfo(string(cpuInfo))
}
