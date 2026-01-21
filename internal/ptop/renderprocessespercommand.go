package ptop

import (
	"fmt"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/themes"
	"github.com/walles/ptop/internal/ui"
)

func renderPerCommand(screen twin.Screen, theme themes.Theme, x0, y0, x1, y1 int, table [][]string, widths []int, commands []commandStats) {
	widths = widths[6:] // Skip the per-process columns

	// Formats are "%5.5s" or "%-5.5s", where "5.5" means "pad and truncate to
	// 5", and the "-" means left-align.
	formatString := fmt.Sprintf("%%-%d.%ds %%%d.%ds %%%d.%ds",
		widths[0], widths[0],
		widths[1], widths[1],
		widths[2], widths[2],
	)

	memoryRamp := ui.NewColorRamp(0.0, 1.0, theme.LoadBarMin(), theme.LoadBarMaxRam())
	cpuRamp := ui.NewColorRamp(0.0, 1.0, theme.LoadBarMin(), theme.LoadBarMaxCpu())

	// +1 = ignore top border
	topBottomRamp := ui.NewColorRamp(float64(y0+1), float64(y1-1), theme.Foreground(), theme.FadedForeground())

	// If y0 = 0 and y1 = 1, then there would be 0 content rows between the
	// borders
	rowsWithoutBorders := y1 - y0 - 1

	maxCpuSecondsPerCommand := 0.0
	maxRssKbPerCommand := 0
	for _, c := range commands {
		if c.cpuTime.Seconds() > maxCpuSecondsPerCommand {
			maxCpuSecondsPerCommand = c.cpuTime.Seconds()
		}
		if c.rssKb > maxRssKbPerCommand {
			maxRssKbPerCommand = c.rssKb
		}
	}

	cpuAndMemBar := ui.NewOverlappingLoadBars(x0+1, x1-1, cpuRamp, memoryRamp)

	//
	// Render table contents
	//

	for rowIndex, row := range table {
		if rowIndex >= rowsWithoutBorders {
			// No more room
			break
		}

		row = row[6:] // Skip the per-process columns
		line := fmt.Sprintf(formatString,
			row[0], row[1], row[2],
		)

		y := y0 + 1 + rowIndex // screen row

		rowStyle := twin.StyleDefault.WithForeground(topBottomRamp.AtInt(y))

		x := x0 + 1 // screen column
		for _, char := range line {
			style := rowStyle
			screen.SetCell(x, y, twin.StyledRune{Rune: char, Style: style})

			if rowIndex < len(commands) {
				command := commands[rowIndex]
				cpuFraction := 0.0
				if maxCpuSecondsPerCommand > 0.0 {
					cpuFraction = command.cpuTime.Seconds() / maxCpuSecondsPerCommand
				}
				memFraction := 0.0
				if maxRssKbPerCommand > 0 {
					memFraction = float64(command.rssKb) / float64(maxRssKbPerCommand)
				}
				cpuAndMemBar.SetCellBackground(screen, x, y, cpuFraction, memFraction)
			}

			x++
		}
	}

	renderFrame(screen, theme, x0, y0, x1, y1, "By Command")
}
