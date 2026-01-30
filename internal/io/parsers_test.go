package io

import (
	"os"
	"testing"

	"github.com/walles/ftop/internal/assert"
)

func TestParseProcNetDev(t *testing.T) {
	exampleBytes, err := os.ReadFile("parsers_test/proc/net/dev.txt")
	assert.Equal(t, err, nil)

	netDevStats, err := parseProcNetDev(string(exampleBytes))
	assert.Equal(t, err, nil)

	// From a Docker container
	assert.Equal(t, netDevStats["eth0 (in)"], 1752)
	assert.Equal(t, netDevStats["eth0 (out)"], 126)
}

func TestParseProcDiskstats(t *testing.T) {
	exampleBytes, err := os.ReadFile("parsers_test/proc/diskstats.txt")
	assert.Equal(t, err, nil)

	diskStats, err := parseProcDiskstats(string(exampleBytes))
	assert.Equal(t, err, nil)

	// From a Docker container
	assert.Equal(t, diskStats["vda1 (read)"], 22602*512)
	assert.Equal(t, diskStats["vda1 (write)"], 10552*512)
}
