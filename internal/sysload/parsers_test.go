package sysload

import (
	"os"
	"testing"

	"github.com/walles/ptop/internal/assert"
)

func TestParseProcCpuInfo(t *testing.T) {
	exampleBytes, err := os.ReadFile("parsers_test/proc/cpuinfo.txt")
	assert.Equal(t, err, nil)

	logical, physical, err := parseProcCpuInfo(string(exampleBytes))
	assert.Equal(t, err, nil)

	// From a Docker container
	assert.Equal(t, logical, 16)
	assert.Equal(t, physical, 16)
}

func TestParseProcMemInfo(t *testing.T) {
	exampleBytes, err := os.ReadFile("parsers_test/proc/meminfo.txt")
	assert.Equal(t, err, nil)

	usedBytes, totalBytes, err := parseProcMemInfo(string(exampleBytes))
	assert.Equal(t, err, nil)

	// Expected values computed from parsers_test/proc/meminfo.txt
	// MemTotal: 6579644 kB -> 6579644 * 1024
	// Used (ram): MemTotal - MemAvailable = 6579644 - 6032168 = 547476 kB
	// Swap used = SwapTotal - SwapFree = 1048572 - 1048572 = 0 kB
	// Total used bytes = (swapused + ramused) * 1024
	assert.Equal(t, totalBytes, uint64(6_737_555_456))
	assert.Equal(t, usedBytes, uint64(560_615_424))
}
