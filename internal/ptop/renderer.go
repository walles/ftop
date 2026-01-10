package ptop

import (
	"time"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/io"
	"github.com/walles/ptop/internal/processes"
)

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

func Render(processesRaw []processes.Process, ioStats []io.Stat, screen twin.Screen) {
	const overviewHeight = 5         // Including borders
	const launchedCommandsHeight = 7 // Including borders

	width, height := screen.Size()
	ioStatsWidth := 25                    // Including borders
	overviewWidth := width - ioStatsWidth // Including borders
	if overviewWidth < 63 {
		overviewWidth = width
		ioStatsWidth = 0
	}

	// Processes use the remaining height. This number includes borders.
	processesHeight := height - overviewHeight - launchedCommandsHeight

	screen.Clear()

	renderOverview(screen, ioStats, overviewWidth)

	// Draw IO stats to the right of the overview...
	if ioStatsWidth > 0 {
		// ... but only when there is room for it.
		renderIoTopList(screen, ioStats, overviewWidth, 0, width-1, 4)
	}

	// -2 to skip the borders
	processesTable, usersHeight, processes, users, commands := createProcessesTable(processesRaw, processesHeight-2)
	renderProcessesBlock(
		screen,
		processesTable,
		processes,
		overviewHeight,
		overviewHeight+processesHeight-1,
		users,
		usersHeight,
		commands,
	)

	renderLaunchedCommands(screen, overviewHeight+processesHeight, height-1)

	screen.Show()
}

func renderOverview(screen twin.Screen, ioStats []io.Stat, overviewWidth int) {
	renderSysload(screen, overviewWidth)
	renderMemoryUsage(screen, overviewWidth)
	renderIOLoad(ioStats, screen, overviewWidth)

	renderFrame(screen, 0, 0, overviewWidth-1, 4, "Overview")
}

func renderFrame(screen twin.Screen, x0, y0, x1, y1 int, title string) {
	colorTitle := twin.NewColorHex(0xffc0c0) // FIXME: Get this from the theme
	colorFrame := twin.NewColorHex(0x7070a0) // FIXME: Get this from the theme

	dividerStyle := twin.StyleDefault.WithForeground(colorFrame)

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
	titleStyle := twin.StyleDefault.WithForeground(colorTitle)
	titleRunes := []rune(title)
	for i, r := range titleRunes {
		if x0+2+i < x1 {
			screen.SetCell(x0+2+i, y0, twin.StyledRune{Rune: r, Style: titleStyle})
		}
	}
}
