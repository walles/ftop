package io

import "os"

func GetNetworkStats() (map[string]uint64, error) {
	procNetDev, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return nil, err
	}

	return parseProcNetDev(string(procNetDev))

}

func GetDiskStats() (map[string]uint64, error) {
	procDiskstats, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		return nil, err
	}

	return parseProcDiskstats(string(procDiskstats))
}
