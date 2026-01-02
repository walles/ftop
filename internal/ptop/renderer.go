package ptop

import (
	"fmt"
	"time"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/processes"
	"github.com/walles/ptop/internal/ui"
)

type userStats struct {
	username     string
	cpuTime      time.Duration
	rssKb        int
	processCount int
}

func Render(processesRaw []processes.Process, screen twin.Screen) {
	_, height := screen.Size()
	screen.Clear()

	renderOverview(screen)

	// 5 = room for the overview section at the top
	prepAndRenderProcesses(processesRaw, screen, 5, height-1)

	screen.Show()
}

func renderOverview(screen twin.Screen) {
	width, _ := screen.Size()

	renderSysload(screen)
	renderMemoryUsage(screen)
	renderIOLoad(screen)

	renderFrame(screen, 0, 0, 4, width-1, "Overview")
}

func renderSysload(screen twin.Screen) {
	column := 2
	for _, char := range "Sysload: 1.4  [8 cores | 16 virtual]  [15m history: ⢸⣿⣿⣿⣿⣿⣿⣷] (this row is fake)" {
		screen.SetCell(column, 1, twin.StyledRune{Rune: char, Style: twin.StyleDefault})
		column++
	}
}

func renderMemoryUsage(screen twin.Screen) {
	column := 2
	for _, char := range "RAM Use: 60%  [19GB / 32GB] (this row is fake)" {
		screen.SetCell(column, 2, twin.StyledRune{Rune: char, Style: twin.StyleDefault})
		column++
	}
}

func renderIOLoad(screen twin.Screen) {
	column := 2
	for _, char := range "IO Load:      [422KB/s / 2781KB/s] disk0 (this row is fake)" {
		screen.SetCell(column, 3, twin.StyledRune{Rune: char, Style: twin.StyleDefault})
		column++
	}
}

// Top and bottom row values are inclusive
func prepAndRenderProcesses(processesRaw []processes.Process, screen twin.Screen, topRow int, bottomRow int) {
	width, _ := screen.Size()
	height := 1 + bottomRow - topRow

	// Decide on section heights
	heightWithoutBorders := height - 2 // 2 = top and bottom frame lines

	// Collect data to show
	heightWithoutHeaders := heightWithoutBorders - 1
	processesByScore := ProcessesByScore(processesRaw)
	if len(processesByScore) > heightWithoutHeaders-1 {
		processesByScore = processesByScore[:heightWithoutHeaders]
	}
	users := UsersByScore(processesRaw)
	if len(users) > heightWithoutHeaders-1 {
		users = users[:heightWithoutHeaders]
	}

	// Figure out column widths
	allInOneTable := toTable(processesByScore, users)
	// 1=left frame, 5=per-process column separators, 2="||", 2=per-user column separators, 1=right frame
	rowSpacing := 1 + 5 + 2 + 2 + 1
	widths := ui.ColumnWidths(allInOneTable, width-rowSpacing)

	perProcessTableWidth := widths[0] + 1 + widths[1] + 1 + widths[2] + 1 + widths[3] + 1 + widths[4] + 1 + widths[5]
	rightPerProcessBorderColumn := perProcessTableWidth + 1 // +1 for the left frame line
	leftPerUserBorderColumn := rightPerProcessBorderColumn + 1
	rightPerUserBorderColumn := leftPerUserBorderColumn + widths[6] + 1 + widths[7] + 1 + widths[8] + 1

	// Render!
	topContentsRow := topRow + 1 // +1 for the top frame line
	doRenderProcesses(allInOneTable, widths, processesByScore, users, screen, topContentsRow, 1)

	renderFrame(screen, topRow, 0, bottomRow, rightPerProcessBorderColumn, "By process")
	renderFrame(screen, topRow, leftPerUserBorderColumn, bottomRow, rightPerUserBorderColumn, "By user")
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

func toTable(processesByScore []processes.Process, usersByScore []userStats) [][]string {
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
				fmt.Sprintf("%d", p.Pid),
				p.Command,
				p.Username,
				p.CpuPercentString(),
				p.CpuTimeString(),
				ui.FormatMemory(int64(p.RssKb)*1024),
			)
		} else {
			// Pad out with empty per-process data
			row = append(row, "", "", "", "", "", "")
		}

		if i < len(usersByScore) {
			u := usersByScore[i]
			row = append(row,
				u.username,
				ui.FormatDuration(u.cpuTime),
				ui.FormatMemory(1024*int64(u.rssKb)),
			)
		} else {
			// Pad out with empty per-user data
			row = append(row, "", "", "")
		}

		table = append(table, row)
	}

	return table
}

func doRenderProcesses(
	table [][]string,
	widths []int,
	processes []processes.Process,
	users []userStats,
	screen twin.Screen,
	firstScreenRow int,
	firstScreenColumn int,
) {
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
		if p.CpuTime != nil && p.CpuTime.Seconds() > maxCpuSecondsPerProcess {
			maxCpuSecondsPerProcess = p.CpuTime.Seconds()
		}
		if p.RssKb > maxRssKbPerProcess {
			maxRssKbPerProcess = p.RssKb
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
					if process.CpuTime != nil && maxCpuSecondsPerProcess > 0.0 {
						cpuFraction = process.CpuTime.Seconds() / maxCpuSecondsPerProcess
					}
					memFraction := 0.0
					if maxRssKbPerProcess > 0 {
						memFraction = float64(process.RssKb) / float64(maxRssKbPerProcess)
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
