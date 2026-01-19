package ptop

import (
	"time"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/io"
	"github.com/walles/ptop/internal/processes"
	"github.com/walles/ptop/internal/themes"
	"github.com/walles/ptop/internal/ui"
)

const minWidth = 80
const minHeight = 22

type stats struct {
	name    string
	cpuTime time.Duration
	rssKb   int
}

type userStats struct {
	stats
}

type commandStats struct {
	stats
}

func Render(screen twin.Screen, theme themes.Theme, processesRaw []processes.Process, ioStats []io.Stat, launches *processes.LaunchNode) {
	const overviewHeight = 5         // Including borders
	const launchedCommandsHeight = 7 // Including borders

	width, height := screen.Size()
	if width < minWidth || height < minHeight {
		renderTooSmallScreen(screen, theme)
		return
	}

	ioStatsWidth := 25                    // Including borders
	overviewWidth := width - ioStatsWidth // Including borders

	// 64 = the width needed for the overview with a double digit number of
	// logical cores.
	if overviewWidth < 64 {
		overviewWidth = width
		ioStatsWidth = 0
	}

	// Processes use the remaining height. This number includes borders.
	processesHeight := height - overviewHeight - launchedCommandsHeight

	screen.Clear()

	renderOverview(screen, theme, ioStats, overviewWidth)

	// Draw IO stats to the right of the overview...
	if ioStatsWidth > 0 {
		// ... but only when there is room for it.
		renderIoTopList(screen, theme, ioStats, overviewWidth, 0, width-1, 4)
	}

	// -2 to skip the borders
	processesTable, usersHeight, processes, users, commands := createProcessesTable(processesRaw, processesHeight-2)
	renderProcessesBlock(
		screen,
		theme,
		processesTable,
		processes,
		overviewHeight,
		overviewHeight+processesHeight-1,
		users,
		usersHeight,
		commands,
	)

	// FIXME: Skip this section if launches is nil. Adjust height based on the
	// height of the rendered graph.
	renderLaunchedCommands(screen, theme, launches, overviewHeight+processesHeight, height-1)

	screen.Show()
}

func renderTooSmallScreen(screen twin.Screen, theme themes.Theme) {
	screen.Clear()

	width, height := screen.Size()
	message := "Make terminal larger"
	if width >= minWidth {
		// Width is fine
		message = "Make terminal taller"
	}
	if height >= minHeight {
		// Height is fine
		message = "Make terminal wider"
	}
	x := (width - len(message)) / 2

	halfWayY := height / 2
	topToMiddleTextRamp := ui.NewColorRamp(0, float64(halfWayY), theme.Background(), theme.Foreground())
	middleToBottomTextRamp := ui.NewColorRamp(float64(halfWayY+1), float64(height), theme.Foreground(), theme.Background())

	for y := range height {
		var color twin.Color
		if y <= halfWayY {
			color = topToMiddleTextRamp.AtInt(y)
		} else {
			color = middleToBottomTextRamp.AtInt(y)
		}
		drawText(screen, x, y, width, message, twin.StyleDefault.WithForeground(color))
	}

	red := twin.NewColorHex(0xff0000)

	if width < minWidth {
		// Red left and right borders
		topMiddleRamp := ui.NewColorRamp(0, float64(halfWayY), theme.Background(), red)
		middleBottomRamp := ui.NewColorRamp(float64(halfWayY+1), float64(height-1), red, theme.Background())

		for y := range height {
			var color twin.Color
			if y <= halfWayY {
				color = topMiddleRamp.AtInt(y)
			} else {
				color = middleBottomRamp.AtInt(y)
			}
			screen.SetCell(0, y, twin.StyledRune{
				Style: twin.StyleDefault.WithForeground(color),
				Rune:  '<',
			})
			screen.SetCell(width-1, y, twin.StyledRune{
				Style: twin.StyleDefault.WithForeground(color),
				Rune:  '>',
			})
		}
	}

	if height < minHeight {
		// Red top and bottom borders
		leftMiddleRamp := ui.NewColorRamp(0, float64(width)/2, theme.Background(), red)
		middleRightRamp := ui.NewColorRamp(float64(width)/2+1, float64(width-1), red, theme.Background())

		for x := range width {
			var color twin.Color
			if x <= width/2 {
				color = leftMiddleRamp.AtInt(x)
			} else {
				color = middleRightRamp.AtInt(x)
			}
			screen.SetCell(x, 0, twin.StyledRune{
				Style: twin.StyleDefault.WithForeground(color),
				Rune:  '^',
			})
			screen.SetCell(x, height-1, twin.StyledRune{
				Style: twin.StyleDefault.WithForeground(color),
				Rune:  'v',
			})
		}
	}

	screen.Show()
}

func renderOverview(screen twin.Screen, theme themes.Theme, ioStats []io.Stat, overviewWidth int) {
	renderSysload(screen, theme, overviewWidth)
	renderMemoryUsage(screen, theme, overviewWidth)
	renderIOLoad(screen, theme, ioStats, overviewWidth)

	renderFrame(screen, theme, 0, 0, overviewWidth-1, 4, "Overview")
}

func renderFrame(screen twin.Screen, theme themes.Theme, x0, y0, x1, y1 int, title string) {
	dividerStyle := twin.StyleDefault.WithForeground(theme.Border())

	for col := x0 + 1; col < x1; col++ {
		screen.SetCell(col, y0, twin.StyledRune{Rune: '─', Style: dividerStyle})
		screen.SetCell(col, y1, twin.StyledRune{Rune: '─', Style: dividerStyle})
	}
	for row := y0 + 1; row < y1; row++ {
		screen.SetCell(x0, row, twin.StyledRune{Rune: '│', Style: dividerStyle})
		screen.SetCell(x1, row, twin.StyledRune{Rune: '│', Style: dividerStyle})
	}
	screen.SetCell(x0, y0, twin.StyledRune{Rune: '┌', Style: dividerStyle})
	screen.SetCell(x1, y0, twin.StyledRune{Rune: '┐', Style: dividerStyle})
	screen.SetCell(x0, y1, twin.StyledRune{Rune: '└', Style: dividerStyle})
	screen.SetCell(x1, y1, twin.StyledRune{Rune: '┘', Style: dividerStyle})

	// Title
	titleStyle := twin.StyleDefault.WithForeground(theme.BorderTitle())
	titleRunes := []rune(title)
	for i, r := range titleRunes {
		if x0+2+i < x1 {
			screen.SetCell(x0+2+i, y0, twin.StyledRune{Rune: r, Style: titleStyle})
		}
	}
}
