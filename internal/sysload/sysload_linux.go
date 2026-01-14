package sysload

import (
	"os"
)

func getMemoryUsage() (usedBytes uint64, totalBytes uint64, err error) {
	memInfo, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}

	return parseProcMemInfo(string(memInfo))
}

func getCpuCoreCounts() (coresLogical int, coresPhysical int, err error) {
	cpuInfo, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 0, 0, err
	}

	return parseProcCpuInfo(string(cpuInfo))
}

func getLoadAverages() (load1M float64, load5M float64, load15M float64, err error) {
	loadAvg, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0, err
	}

	return parseProcLoadAvg(string(loadAvg))
}
