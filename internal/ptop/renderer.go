package ptop

import (
	"time"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/io"
	"github.com/walles/ptop/internal/processes"
)

type userStats struct {
	username     string
	cpuTime      time.Duration
	rssKb        int
	processCount int
}

func Render(processesRaw []processes.Process, ioStats []io.Stat, screen twin.Screen) {
	const overviewHeight = 5         // Including borders
	const launchedBinariesHeight = 7 // Including borders

	width, height := screen.Size()
	ioStatsWidth := 25                    // Including borders
	overviewWidth := width - ioStatsWidth // Including borders
	if overviewWidth < 63 {
		overviewWidth = width
		ioStatsWidth = 0
	}

	// Processes use the remaining height
	processesHeight := height - overviewHeight - launchedBinariesHeight

	screen.Clear()

	renderOverview(screen, ioStats, overviewWidth)

	// Draw IO stats to the right of the overview...
	if ioStatsWidth > 0 {
		// ... but only when there is room for it.
		renderIoTopList(screen, ioStats, overviewWidth, width-1)
	}

	// 5 = room for the overview section at the top
	prepAndRenderProcesses(processesRaw, screen, overviewHeight, overviewHeight+processesHeight-1)

	renderLaunchedBinaries(screen, overviewHeight+processesHeight, height-1)

	screen.Show()
}

func renderOverview(screen twin.Screen, ioStats []io.Stat, overviewWidth int) {
	renderSysload(screen, overviewWidth)
	renderMemoryUsage(screen, overviewWidth)
	renderIOLoad(ioStats, screen, overviewWidth)

	renderFrame(screen, 0, 0, 4, overviewWidth-1, "Overview")
}

func renderFrame(screen twin.Screen, topRow int, leftColumn int, bottomRow int, rightColumn int, title string) {
	colorTitle := twin.NewColorHex(0xffc0c0) // FIXME: Get this from the theme
	colorFrame := twin.NewColorHex(0x7070a0) // FIXME: Get this from the theme

	dividerStyle := twin.StyleDefault.WithForeground(colorFrame)

	for col := leftColumn + 1; col < rightColumn; col++ {
		screen.SetCell(col, topRow, twin.StyledRune{Rune: '─', Style: dividerStyle})
		screen.SetCell(col, bottomRow, twin.StyledRune{Rune: '─', Style: dividerStyle})
	}
	for row := topRow + 1; row < bottomRow; row++ {
		screen.SetCell(leftColumn, row, twin.StyledRune{Rune: '│', Style: dividerStyle})
		screen.SetCell(rightColumn, row, twin.StyledRune{Rune: '│', Style: dividerStyle})
	}
	screen.SetCell(leftColumn, topRow, twin.StyledRune{Rune: '┌', Style: dividerStyle})
	screen.SetCell(rightColumn, topRow, twin.StyledRune{Rune: '┐', Style: dividerStyle})
	screen.SetCell(leftColumn, bottomRow, twin.StyledRune{Rune: '└', Style: dividerStyle})
	screen.SetCell(rightColumn, bottomRow, twin.StyledRune{Rune: '┘', Style: dividerStyle})

	// Title
	titleStyle := twin.StyleDefault.WithForeground(colorTitle)
	titleRunes := []rune(title)
	for i, r := range titleRunes {
		if leftColumn+2+i < rightColumn {
			screen.SetCell(leftColumn+2+i, topRow, twin.StyledRune{Rune: r, Style: titleStyle})
		}
	}
}
