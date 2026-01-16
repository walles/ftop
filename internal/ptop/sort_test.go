package ptop

import (
	"testing"
	"time"

	"github.com/walles/ptop/internal/assert"
	"github.com/walles/ptop/internal/processes"
)

func TestProcessesByScore(t *testing.T) {
	procs := []processes.Process{
		{RssKb: 300},
		{RssKb: 100},
		{RssKb: 200},
	}

	sorted := SortByScore(procs, func(p processes.Process) stats {
		return stats{
			rssKb: p.RssKb,
		}
	})

	assert.Equal(t, sorted[0].RssKb, 300)
	assert.Equal(t, sorted[1].RssKb, 200)
	assert.Equal(t, sorted[2].RssKb, 100)
}

// If one process has max mem and the other max CPU, always put the CPU one on
// top. This is because I always want the top CPU process at the top of the
// list, since that's what I would expect from top.
func TestProcessesByScore_TieBreak_CPUOverMem(t *testing.T) {
	muchCpuLittleMem := processes.Process{
		Command: "most CPU",
		RssKb:   100,
		CpuTime: toDurationPointer(300 * time.Second),
	}
	muchMemAboveAverageCpu := processes.Process{
		Command: "most mem",
		RssKb:   300,
		CpuTime: toDurationPointer(200 * time.Second),
	}

	sorted := SortByScore([]processes.Process{muchMemAboveAverageCpu, muchCpuLittleMem}, func(p processes.Process) stats {
		return stats{
			rssKb:   p.RssKb,
			cpuTime: *p.CpuTime,
		}
	})

	assert.Equal(t, sorted[0].Command, "most CPU")
	assert.Equal(t, sorted[1].Command, "most mem")

	// Try the other direction
	sorted = SortByScore([]processes.Process{muchCpuLittleMem, muchMemAboveAverageCpu}, func(p processes.Process) stats {
		return stats{
			rssKb:   p.RssKb,
			cpuTime: *p.CpuTime,
		}
	})

	assert.Equal(t, sorted[0].Command, "most CPU")
	assert.Equal(t, sorted[1].Command, "most mem")
}

func toDurationPointer(d time.Duration) *time.Duration {
	return &d
}
