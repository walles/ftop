package processes

import (
	"fmt"
	"time"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/ui"
)

// Render the given processes to the given screen
func Render(processes []Process, screen twin.Screen) {
	width, height := screen.Size()

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

	// "-5" = the number of between-column-spaces we need
	widths := ui.ColumnWidths(table, width-5)

	// Formats are "%5.5s" or "%-5.5s", where "5.5" means "pad and truncate to
	// 5", and the "-" means left-align.
	formatString := fmt.Sprintf("%%%d.%ds %%-%d.%ds %%-%d.%ds %%%d.%ds %%%d.%ds %%%d.%ds",
		widths[0], widths[0],
		widths[1], widths[1],
		widths[2], widths[2],
		widths[3], widths[3],
		widths[4], widths[4],
		widths[5], widths[5],
	)

	colorBg := twin.NewColor24Bit(0, 0, 0) // FIXME: Get this fallback from the theme
	if screen.TerminalBackground() != nil {
		colorBg = *screen.TerminalBackground()
	}

	colorTop := twin.NewColorHex(0xffffff) // FIXME: Get this from the theme
	colorBottom := colorTop.Mix(colorBg, 0.66)
	// 1.0 = ignore the header line
	topBottomRamp := ui.NewColorRamp(colorTop, colorBottom, 1.0, float64(len(table)-1))

	maxCpuTime := time.Duration(0)
	for _, p := range processes {
		if p.cpuTime != nil && *p.cpuTime > maxCpuTime {
			maxCpuTime = *p.cpuTime
		}
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
			style = twin.StyleDefault
			style = style.WithForeground(topBottomRamp.AtInt(rowIndex))
		}

		for x, char := range line {
			screen.SetCell(x, rowIndex, twin.StyledRune{Rune: char, Style: style})
		}
	}

	screen.Show()
}
