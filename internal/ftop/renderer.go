package ftop

import (
	"time"

	"github.com/walles/ftop/internal/io"
	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

// Found through experimentation, all narrow columns fit at this width when I
// tried it.
const minWidth = 50

const minHeight = 11

type stats struct {
	name     string
	cpuTime  time.Duration
	nativity uint
	rssKb    int
}

type userStats struct {
	stats
}

type commandStats struct {
	stats
}

func (u *Ui) Render(processesRaw []processes.Process, ioStats []io.Stat, launches *processes.LaunchNode) {
	const overviewHeight = 5 // Including borders

	width, height := u.screen.Size()
	if width < minWidth || height < minHeight {
		renderTooSmallScreen(u.screen, u.theme)
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

	heightWithoutOverview := height - overviewHeight
	maxScreenLaunchedCommandsHeight := heightWithoutOverview / 3      // Including borders
	launchedCommandsHeight := getLaunchedCommandsHeight(launches) + 2 // + 2 for borders
	if launchedCommandsHeight > maxScreenLaunchedCommandsHeight {
		launchedCommandsHeight = maxScreenLaunchedCommandsHeight
	}
	if launchedCommandsHeight <= 2 {
		launchedCommandsHeight = 0
	}

	// Processes use the remaining height. This number includes borders.
	processesHeight := height - overviewHeight - launchedCommandsHeight
	if processesHeight < 6 {
		// 6 = Heights of per-user and per-command blocks with one line each and
		// borders. From top to bottom: border, user, border, border, command,
		// border.
		launchedCommandsHeight = 0
		processesHeight = height - overviewHeight
	}

	processesBottomRow := overviewHeight + processesHeight - 1

	u.screen.Clear()

	renderOverview(u.screen, u.theme, ioStats, overviewWidth)

	// Draw IO stats to the right of the overview...
	if ioStatsWidth > 0 {
		// ... but only when there is room for it.
		renderIoTopList(u.screen, u.theme, ioStats, overviewWidth, 0, width-1, 4)
	}

	if width < u.minThreePanesScreenWidth {
		u.renderSingleProcessesPane(processesRaw, overviewHeight, processesBottomRow)
	} else if u.canRenderThreeProcessPanes(u.screen, processesRaw, overviewHeight, processesBottomRow) {
		u.renderThreeProcessPanes(processesRaw, overviewHeight, processesBottomRow)
	} else {
		u.renderSingleProcessesPane(processesRaw, overviewHeight, processesBottomRow)

		// Current width didn't work, maybe one column more would do the trick?
		newMinThreePanesWidth := width + 1
		if newMinThreePanesWidth > u.minThreePanesScreenWidth {
			log.Debugf("Bumping minThreePanesScreenWidth from %d to %d...", u.minThreePanesScreenWidth, newMinThreePanesWidth)
			u.minThreePanesScreenWidth = newMinThreePanesWidth
		}
	}

	if launchedCommandsHeight > 0 {
		renderLaunchedCommands(u.screen, u.theme, launches, processesBottomRow+1, height-1)
	}

	u.screen.Show()
}

func renderOverview(screen twin.Screen, theme themes.Theme, ioStats []io.Stat, overviewWidth int) {
	renderSysload(screen, theme, overviewWidth)
	renderMemoryUsage(screen, theme, overviewWidth)
	renderIOLoad(screen, theme, ioStats, overviewWidth)

	renderFrame(screen, theme, 0, 0, overviewWidth-1, 4, "Overview")

	// Draw "Quit" prompt in upper right corner
	width, _ := screen.Size()
	x := width - (len("Quit") + 2)
	x += screen.SetCell(x, 0, twin.StyledRune{Rune: 'Q', Style: theme.PromptKey()})
	drawText(screen, x, 0, overviewWidth-1, "uit", theme.PromptActive())
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
