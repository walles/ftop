package processes

import (
	"fmt"
	"time"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/ui"
)

// Render the given processes to the given screen
func Render(processes []Process, screen twin.Screen) {
	_, height := screen.Size()

	screen.Clear()

	table := [][]string{
		{"PID", "COMMAND", "USERNAME", "CPU", "CPUTIME", "RAM"},
	}
	for i, p := range processes {
		if i >= height {
			break
		}

		table = append(table, []string{
			fmt.Sprintf("%d", p.pid),
			p.command,
			p.username,
			p.CpuPercentString(),
			p.CpuTimeString(),
			p.RamPercentString(),
		})
	}

	widths := ui.ColumnWidths(table)
	formatString := fmt.Sprintf("%%%ds %%-%ds %%-%ds %%%ds %%%ds %%%ds",
		widths[0], widths[1], widths[2], widths[3], widths[4], widths[5],
	)

	bgColor := twin.NewColor24Bit(0, 0, 0) // FIXME: Get this fallback from the theme
	if screen.TerminalBackground() != nil {
		bgColor = *screen.TerminalBackground()
	}

	maxCpuTime := time.Duration(0)
	for _, p := range processes {
		if p.cpuTime != nil && *p.cpuTime > maxCpuTime {
			maxCpuTime = *p.cpuTime
		}
	}

	colorHot := twin.NewColorHex(0xff8888)  // FIXME: Get this from the theme
	colorCold := twin.NewColorHex(0xffffff) // FIXME: Get this from the theme
	var heatRamp ui.ColorRamp
	if maxCpuTime == 0 {
		// All-cold ramp when all times are zero
		heatRamp = ui.NewColorRamp(colorCold, colorCold, 0.0, 1.0)
	} else {
		// Show everything below this threshold as all cold
		allCold := maxCpuTime.Seconds() * 0.25
		heatRamp = ui.NewColorRamp(colorCold, colorHot, allCold, maxCpuTime.Seconds())
	}

	for rowIndex, row := range table {
		line := fmt.Sprintf(formatString,
			row[0], row[1], row[2], row[3], row[4], row[5],
		)

		var style twin.Style
		if rowIndex == 0 {
			// Header row, header style
			style = twin.StyleDefault.WithAttr(twin.AttrBold)
		} else {
			temperatureColor := heatRamp.AtValue(processes[rowIndex-1].cpuTime.Seconds())

			// Fade towards this color rather than the background, we want the
			// last lines to still be visible
			colorMin := temperatureColor.Mix(bgColor, 0.5)

			// 1.0 means ignoring the header line when picking the color
			ramp := ui.NewColorRamp(temperatureColor, colorMin, 1.0, float64(len(table)-1))
			style = twin.StyleDefault.WithForeground(ramp.AtInt(rowIndex))
		}

		for x, char := range line {
			screen.SetCell(x, rowIndex, twin.StyledRune{Rune: char, Style: style})
		}
	}

	screen.Show()
}
