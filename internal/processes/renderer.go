package processes

import (
	"fmt"
	"time"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/ui"
)

type userStats struct {
	username     string
	cpuTime      time.Duration
	rssKb        int
	processCount int
}

func Render(processesRaw []Process, screen twin.Screen) {
	width, height := screen.Size()

	// Decide on section heights
	heightWithoutPadding := height - 2 // 2 = top and bottom frame lines

	// Decide on section contents. "-1" = Leave room for the header row
	processesByScore := ProcessesByScore(processesRaw)
	if len(processesByScore) > heightWithoutPadding-1 {
		processesByScore = processesByScore[:heightWithoutPadding-1]
	}
	users := UsersByScore(processesRaw)
	if len(users) > heightWithoutPadding-1 {
		users = users[:heightWithoutPadding-1]
	}

	// Adjust heights to what we actually have
	heightWithoutPadding = max(len(processesByScore), len(users))

	// Figure out column widths
	allInOneTable := toTable(processesByScore, users)
	widths := ui.ColumnWidths(allInOneTable, width-4) // 4 = left and right frame around each section

	perProcessTableWidth := widths[0] + 1 + widths[1] + 1 + widths[2] + 1 + widths[3] + 1 + widths[4] + 1 + widths[5]
	rightPerProcessBorderColumn := perProcessTableWidth + 1
	leftPerUserBorderColumn := rightPerProcessBorderColumn + 1
	rightPerUserBorderColumn := leftPerUserBorderColumn + widths[6] + 1 + widths[7] + 1 + widths[8] + 1

	// Render!
	screen.Clear()
	renderSection(allInOneTable, widths, processesByScore, users, screen, 1, 1)

	bottomBorderRow := heightWithoutPadding + 2
	renderFrame(screen, 0, 0, bottomBorderRow, rightPerProcessBorderColumn, "By process")
	renderFrame(screen, 0, leftPerUserBorderColumn, bottomBorderRow, rightPerUserBorderColumn, "By user")

	screen.Show()
}

func renderFrame(screen twin.Screen, topRow int, leftColumn int, bottomRow int, rightColumn int, title string) {
	colorTitle := twin.NewColorHex(0xffc0c0)   // FIXME: Get this from the theme
	colorDivider := twin.NewColorHex(0x7070a0) // FIXME: Get this from the theme

	dividerStyle := twin.StyleDefault.WithForeground(colorDivider)

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

func toTable(processesByScore []Process, usersByScore []userStats) [][]string {
	headerLine := []string{
		// These first ones are for the per-process table
		"PID", "Command", "User name", "CPU", "Time", "RAM",
		// These columns are for the per-user table
		"User name", "CPU", "RAM",
	}

	var table [][]string

	// Header line
	table = append(table, headerLine)

	for i := 0; i < max(len(processesByScore), len(usersByScore)); i++ {
		row := make([]string, 0, len(headerLine))

		if i < len(processesByScore) {
			p := processesByScore[i]
			row = append(row,
				fmt.Sprintf("%d", p.pid),
				p.command,
				p.username,
				p.CpuPercentString(),
				p.CpuTimeString(),
				formatMemory(int64(p.rssKb)*1024),
			)
		} else {
			// Pad out with empty per-process data
			row = append(row, "", "", "", "", "", "")
		}

		if i < len(usersByScore) {
			u := usersByScore[i]
			row = append(row,
				u.username,
				formatDuration(u.cpuTime),
				formatMemory(1024*int64(u.rssKb)),
			)
		} else {
			// Pad out with empty per-user data
			row = append(row, "", "", "")
		}

		table = append(table, row)
	}

	return table
}

func renderSection(table [][]string, widths []int, processes []Process, users []userStats, screen twin.Screen, firstScreenRow int, firstScreenColumn int) {
	// Formats are "%5.5s" or "%-5.5s", where "5.5" means "pad and truncate to
	// 5", and the "-" means left-align.
	formatString := fmt.Sprintf("%%%d.%ds %%-%d.%ds %%-%d.%ds %%%d.%ds %%%d.%ds %%%d.%ds||%%-%d.%ds %%%d.%ds %%%d.%ds",
		widths[0], widths[0],
		widths[1], widths[1],
		widths[2], widths[2],
		widths[3], widths[3],
		widths[4], widths[4],
		widths[5], widths[5],
		widths[6], widths[6],
		widths[7], widths[7],
		widths[8], widths[8],
	)

	// NOTE: Use some online OKLCH color picker for experimenting with colors
	colorLoadBarMin := twin.NewColorHex(0x000000)    // FIXME: Get this from the theme
	colorLoadBarMaxRAM := twin.NewColorHex(0x2020ff) // FIXME: Get this from the theme
	colorLoadBarMaxCPU := twin.NewColorHex(0x801020) // FIXME: Get this from the theme
	memoryRamp := ui.NewColorRamp(0.0, 1.0, colorLoadBarMin, colorLoadBarMaxRAM)
	cpuRamp := ui.NewColorRamp(0.0, 1.0, colorLoadBarMin, colorLoadBarMaxCPU)

	colorBg := twin.NewColor24Bit(0, 0, 0) // FIXME: Get this fallback from the theme
	if screen.TerminalBackground() != nil {
		colorBg = *screen.TerminalBackground()
	}

	colorTop := twin.NewColorHex(0xdddddd) // FIXME: Get this from the theme
	colorBottom := colorTop.Mix(colorBg, 0.66)
	// 1.0 = ignore the header line
	topBottomRamp := ui.NewColorRamp(1.0, float64(len(table)-1), colorTop, colorBottom)

	perProcessTableWidth := widths[0] + 1 + widths[1] + 1 + widths[2] + 1 + widths[3] + 1 + widths[4] + 1 + widths[5]

	perUserTableWidth := widths[6] + 1 + widths[7] + 1 + widths[8]
	perUserTableStart := perProcessTableWidth + 2 // +2 for the "||" divider

	maxCpuSecondsPerProcess := 0.0
	maxRssKbPerProcess := 0
	for _, p := range processes {
		if p.cpuTime != nil && p.cpuTime.Seconds() > maxCpuSecondsPerProcess {
			maxCpuSecondsPerProcess = p.cpuTime.Seconds()
		}
		if p.rssKb > maxRssKbPerProcess {
			maxRssKbPerProcess = p.rssKb
		}
	}

	maxCpuSecondsPerUser := 0.0
	maxRssKbPerUser := 0
	for _, u := range users {
		cpuSeconds := u.cpuTime.Seconds()
		if cpuSeconds > maxCpuSecondsPerUser {
			maxCpuSecondsPerUser = cpuSeconds
		}
		if u.rssKb > maxRssKbPerUser {
			maxRssKbPerUser = u.rssKb
		}
	}

	perProcessCpuBar := ui.NewLoadBar(0, perProcessTableWidth-1, cpuRamp, colorBg)
	perProcessMemBar := ui.NewLoadBar(0, perProcessTableWidth-1, memoryRamp, colorBg)

	perUserCpuBar := ui.NewLoadBar(perUserTableStart, perUserTableStart+perUserTableWidth-1, cpuRamp, colorBg)
	perUserMemBar := ui.NewLoadBar(perUserTableStart, perUserTableStart+perUserTableWidth-1, memoryRamp, colorBg)

	for rowIndex, row := range table {
		line := fmt.Sprintf(formatString,
			row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[8],
		)

		var rowStyle twin.Style
		if rowIndex == 0 {
			// Header row, header style
			rowStyle = twin.StyleDefault.WithAttr(twin.AttrBold)
		} else {
			rowStyle = twin.StyleDefault
			rowStyle = rowStyle.WithForeground(topBottomRamp.AtInt(rowIndex))
		}

		// x is relative to the left edge of the table, not to the screen
		x := 0

		for _, char := range line {
			style := rowStyle
			if rowIndex == 0 {
				// Header row, no load bars here
			} else {
				index := rowIndex - 1 // Because rowIndex 0 is the header
				if index < len(processes) {
					process := processes[index]
					cpuFraction := 0.0
					if process.cpuTime != nil && maxCpuSecondsPerProcess > 0.0 {
						cpuFraction = process.cpuTime.Seconds() / maxCpuSecondsPerProcess
					}
					memFraction := 0.0
					if maxRssKbPerProcess > 0 {
						memFraction = float64(process.rssKb) / float64(maxRssKbPerProcess)
					}
					if cpuFraction > memFraction {
						// Draw memory last so it ends up in front of CPU
						perProcessCpuBar.SetBgColor(&style, x, cpuFraction, true)
						perProcessMemBar.SetBgColor(&style, x, memFraction, false)
					} else {
						// Draw CPU last so it ends up in front of memory
						perProcessMemBar.SetBgColor(&style, x, memFraction, true)
						perProcessCpuBar.SetBgColor(&style, x, cpuFraction, false)
					}
				}
				if index < len(users) {
					user := users[index]
					cpuFraction := 0.0
					if maxCpuSecondsPerUser > 0.0 {
						cpuFraction = user.cpuTime.Seconds() / maxCpuSecondsPerUser
					}
					memFraction := 0.0
					if maxRssKbPerUser > 0 {
						memFraction = float64(user.rssKb) / float64(maxRssKbPerUser)
					}
					if cpuFraction > memFraction {
						// Draw memory last so it ends up in front of CPU
						perUserCpuBar.SetBgColor(&style, x, cpuFraction, true)
						perUserMemBar.SetBgColor(&style, x, memFraction, false)
					} else {
						// Draw CPU last so it ends up in front of memory
						perUserMemBar.SetBgColor(&style, x, memFraction, true)
						perUserCpuBar.SetBgColor(&style, x, cpuFraction, false)
					}
				}
			}
			screen.SetCell(firstScreenColumn+x, firstScreenRow+rowIndex, twin.StyledRune{Rune: char, Style: style})
			x++
		}
	}
}
