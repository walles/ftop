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
