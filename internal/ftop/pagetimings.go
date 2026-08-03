package ftop

import (
	"fmt"
	"time"

	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/util"
)

func (u *Ui) timingsForPaging(proc *processes.Process, pt *pageText) {
	pt.writeTitle("Timings")

	age := time.Since(proc.StartTime())
	cpuTime := time.Duration(0)
	if proc.CpuTimeTotal != nil {
		cpuTime = *proc.CpuTimeTotal
	}
	percentCpu := 100.0 * float64(cpuTime) / float64(age)

	pt.writeLine(fmt.Sprintf(
		"Started %s ago at %s. It used %s CPU, or %s.",
		u.highlight(util.FormatDuration(age)),
		u.highlight(proc.StartTime().Format(DISPLAY_TIME_FORMAT)),
		u.highlight(util.FormatDuration(cpuTime)),
		u.highlight(util.FormatPercent(percentCpu)),
	))
}
