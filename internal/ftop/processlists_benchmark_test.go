package ftop

import (
	"fmt"
	"testing"
	"time"

	"github.com/walles/ftop/internal/processes"
)

var benchmarkSortProcessesForDisplaySink []processes.Process

func makeBenchmarkProcesses(count int) []processes.Process {
	processesRaw := make([]processes.Process, 0, count)
	for i := range count {
		cpuSeconds := (i*37)%5000 + 1
		cpuTime := time.Duration(cpuSeconds) * time.Second

		process := processes.Process{
			Pid:      1000 + i,
			Cmdline:  fmt.Sprintf("bench-cmd-%d", i%40),
			Username: "bench-user",
			RssKb:    ((i * 73) % 900000) + 512,
			CpuTime:  &cpuTime,
			Nativity: (i * 19) % 200,
		}

		processesRaw = append(processesRaw, process)
	}

	return processesRaw
}

func BenchmarkSortProcessesForDisplay(b *testing.B) {
	processesRaw := makeBenchmarkProcesses(1000)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		benchmarkSortProcessesForDisplaySink = sortProcessesForDisplay(processesRaw)
	}
}
