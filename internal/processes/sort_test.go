package processes

import (
	"testing"

	"github.com/walles/ptop/internal/assert"
)

func TestProcessesByScore(t *testing.T) {
	processes := []Process{
		{rssKb: 300},
		{rssKb: 100},
		{rssKb: 200},
	}

	sorted := ProcessesByScore(processes)
	assert.Equal(t, sorted[0].rssKb, 300)
	assert.Equal(t, sorted[1].rssKb, 200)
	assert.Equal(t, sorted[2].rssKb, 100)
}
