package ftop

import (
	"fmt"
	"time"

	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/ui"
	"github.com/walles/ftop/internal/util"
	"github.com/walles/twin"
)

func (u *Ui) closeLaunchesForPaging(proc *processes.Process, pt *pageText) {
	pt.writeTitle("Other Processes Launched Close To " + proc.String())

	procs := processes.CloseLaunches(proc)
	if len(procs) == 0 {
		pt.writeLine("<Nothing else was launched close to this process>")
		return
	}

	zero := proc.StartTime()

	delta := func(p *processes.Process) time.Duration {
		if p.StartTime().Before(zero) {
			return zero.Sub(p.StartTime())
		}

		return p.StartTime().Sub(zero)
	}

	maxDelta := delta(procs[0])
	minDelta := delta(procs[0])
	for _, p := range procs {
		d := delta(p)
		if d > maxDelta {
			maxDelta = d
		}
		if d < minDelta {
			minDelta = d
		}
	}

	ramp := ui.NewColorRamp(
		float64(minDelta.Milliseconds()),
		float64(maxDelta.Milliseconds()),
		u.theme.Foreground(),
		u.theme.FadedForeground(),
	)

	for _, p := range procs {
		beforeOrAfter := "after"
		deltaT := p.StartTime().Sub(zero).Abs()
		if p.StartTime().Before(zero) {
			beforeOrAfter = "before"
		}

		deltaString := util.FormatDuration(deltaT) + " " + beforeOrAfter
		if deltaT.Milliseconds() == 0 {
			deltaString = "at the same time as"
		}

		style := twin.StyleDefault.WithForeground(ramp.AtValue(float64(deltaT.Milliseconds())))
		highlighted := twin.StyleDefault.WithForeground(u.theme.HighlightedForeground())
		pt.writeLine(fmt.Sprintf("%s%s launched %s%s%s %s",
			style.RenderUpdateFrom(twin.StyleDefault, twin.ColorCount24bit),
			p.String(),
			highlighted.RenderUpdateFrom(style, twin.ColorCount24bit),
			deltaString,
			twin.StyleDefault.RenderUpdateFrom(highlighted, twin.ColorCount24bit),
			proc.String(),
		))
	}
}
