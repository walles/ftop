package processes

import (
	"fmt"

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

	colorMin := twin.NewColor24Bit(255, 255, 255) // FIXME: Get this from the theme
	bgColor := twin.NewColor24Bit(0, 0, 0)        // FIXME: Get this fallback from the theme
	if screen.TerminalBackground() != nil {
		bgColor = *screen.TerminalBackground()
	}
	colorMax := bgColor.Mix(colorMin, 0.5)
	ramp := ui.NewColorRamp(
		colorMin,
		colorMax,
		1.0, // Skip header row 0, it doesn't need coloring
		float64(len(table)-1),
	)

	for rowIndex, row := range table {
		line := fmt.Sprintf(formatString,
			row[0], row[1], row[2], row[3], row[4], row[5],
		)

		var style twin.Style
		if rowIndex == 0 {
			// Header row, header style
			style = twin.StyleDefault.WithAttr(twin.AttrBold)
		} else {
			style = twin.StyleDefault.WithForeground(ramp.AtInt(rowIndex))
		}

		for x, char := range line {
			screen.SetCell(x, rowIndex, twin.StyledRune{Rune: char, Style: style})
		}
	}

	screen.Show()
}
