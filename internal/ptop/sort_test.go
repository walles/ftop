package ptop

import (
	"testing"

	"github.com/walles/ptop/internal/assert"
	"github.com/walles/ptop/internal/processes"
)

func TestProcessesByScore(t *testing.T) {
	processes := []processes.Process{
		{RssKb: 300},
		{RssKb: 100},
		{RssKb: 200},
	}

	sorted := ProcessesByScore(processes)
	assert.Equal(t, sorted[0].RssKb, 300)
	assert.Equal(t, sorted[1].RssKb, 200)
	assert.Equal(t, sorted[2].RssKb, 100)
}
