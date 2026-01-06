package ptop

import (
	"fmt"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/processes"
	"github.com/walles/ptop/internal/ui"
)

// Top and bottom row values are inclusive
func prepAndRenderProcesses(processesRaw []processes.Process, screen twin.Screen, topRow int, bottomRow int) {
	width, _ := screen.Size()
	height := 1 + bottomRow - topRow

	// Decide on section heights
	processesHeightWithoutBorders := height - 2 // 2 = top and bottom frame lines

	// The -1s are for the separator lines between sections
	usersHeightWithoutBorders := processesHeightWithoutBorders/2 - 1
	//binariesHeightWithoutBorders := processesHeightWithoutBorders - usersHeightWithoutBorders - 1

	// Collect data to show
	procsHeightWithoutHeaders := processesHeightWithoutBorders - 1
	processesByScore := ProcessesByScore(processesRaw)
	if len(processesByScore) > procsHeightWithoutHeaders-1 {
		processesByScore = processesByScore[:procsHeightWithoutHeaders]
	}

	usersHeightWithoutHeaders := usersHeightWithoutBorders - 1
	users := UsersByScore(processesRaw)
	if len(users) > usersHeightWithoutHeaders-1 {
		users = users[:usersHeightWithoutHeaders]
	}

	/*
		binariesHeightWithoutHeaders := binariesHeightWithoutBorders - 1
		binaries := BinariesByScore(processesRaw)
		if len(binaries) > binariesHeightWithoutHeaders-1 {
			binaries = binaries[:binariesHeightWithoutHeaders]
		}
	*/

	// Figure out column widths
	allInOneTable := toTable(processesByScore, users)
	// 1=left frame, 5=per-process column separators, 2="||", 2=per-user column separators, 1=right frame
	rowSpacing := 1 + 5 + 2 + 2 + 1
	widths := ui.ColumnWidths(allInOneTable, width-rowSpacing, false) // Don't grow the PID column, that looks weird

	perProcessTableWidth := widths[0] + 1 + widths[1] + 1 + widths[2] + 1 + widths[3] + 1 + widths[4] + 1 + widths[5]
	rightPerProcessBorderColumn := perProcessTableWidth + 1 // +1 for the left frame line
	leftPerUserBorderColumn := rightPerProcessBorderColumn + 1
	rightPerUserBorderColumn := leftPerUserBorderColumn + widths[6] + 1 + widths[7] + 1 + widths[8] + 1

	// Render!
	topContentsRow := topRow + 1 // +1 for the top frame line
	doRenderProcesses(allInOneTable, widths, processesByScore, users, screen, topContentsRow, 1)

	renderFrame(screen, topRow, 0, bottomRow, rightPerProcessBorderColumn, "By process")
	renderLegend(screen, bottomRow, rightPerProcessBorderColumn)

	usersBottomBorder := topRow + 1 + usersHeightWithoutBorders
	renderFrame(screen, topRow, leftPerUserBorderColumn, usersBottomBorder, rightPerUserBorderColumn, "By user")

	binariesTopRow := usersBottomBorder + 1
	binariesBottomRow := bottomRow
	renderFrame(screen, binariesTopRow, leftPerUserBorderColumn, binariesBottomRow, rightPerUserBorderColumn, "By binary")
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

	perProcessCpuAndMemBar := ui.NewOverlappingLoadBars(
		firstScreenColumn, firstScreenColumn+perProcessTableWidth-1, cpuRamp, memoryRamp)
	perUserCpuAndMemBar := ui.NewOverlappingLoadBars(
		firstScreenColumn+perUserTableStart, firstScreenColumn+perUserTableStart+perUserTableWidth-1, cpuRamp, memoryRamp)

	processUserColumn0 := firstScreenColumn + widths[0] + 1 + widths[1] + 1
	processUserColumnN := processUserColumn0 + widths[2] - 1
	userUserColumn0 := firstScreenColumn + perUserTableStart
	userUserColumnN := userUserColumn0 + widths[6] - 1
	currentUsername := getCurrentUsername()

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
		x := firstScreenColumn

		for _, char := range line {
			style := rowStyle
			if x >= processUserColumn0 && x <= processUserColumnN {
				username := row[2]
				if username == "root" && currentUsername != "root" {
					style = style.WithAttr(twin.AttrDim)
				} else if username != currentUsername {
					style = style.WithAttr(twin.AttrBold)
				}
			}

			screen.SetCell(x, firstScreenRow+rowIndex, twin.StyledRune{Rune: char, Style: style})

			if rowIndex == 0 {
				// Header row, no load bars here
				x++
				continue
			}

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
				perProcessCpuAndMemBar.SetCellBackground(screen, x, firstScreenRow+rowIndex, cpuFraction, memFraction)
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
				perUserCpuAndMemBar.SetCellBackground(screen, x, firstScreenRow+rowIndex, cpuFraction, memFraction)

				// Bold the current username
				if user.username == currentUsername && x >= userUserColumn0 && x <= userUserColumnN {
					cell := screen.GetCell(x, firstScreenRow+rowIndex)
					cell.Style = cell.Style.WithAttr(twin.AttrBold)
					screen.SetCell(x, firstScreenRow+rowIndex, cell)
				}
			}

			x++
		}
	}
}

func toTable(processesByScore []processes.Process, usersByScore []userStats) [][]string {
	headerLine := []string{
		// These first ones are for the per-process table
		"PID", "Command", "Username", "CPU", "Time", "RAM",
		// These columns are for the per-user table
		"Username", "CPU", "RAM",
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

// Towards the right, draw "CPU" with a CPU load bar behind it, and "RAM" with a
// RAM load bar behind it.
func renderLegend(screen twin.Screen, row int, rightmostFrameBorder int) {
	colorText := twin.NewColorHex(0x7070a0)       // FIXME: Get this from the theme. Same as the frame color.
	colorLoadBarMin := twin.NewColorHex(0x000000) // FIXME: Get this from the theme

	// Turn up the bottom color this much so it's visible in the small legend
	const adjustUp = 0.6

	colorLoadBarMaxRAM := twin.NewColorHex(0x2020ff) // FIXME: Get this from the theme
	colorLoadBarMinRAM := colorLoadBarMin.Mix(colorLoadBarMaxRAM, adjustUp)

	colorLoadBarMaxCPU := twin.NewColorHex(0x801020) // FIXME: Get this from the theme
	colorLoadBarMinCPU := colorLoadBarMin.Mix(colorLoadBarMaxCPU, adjustUp)

	memoryRamp := ui.NewColorRamp(0.0, 1.0, colorLoadBarMinRAM, colorLoadBarMaxRAM)
	cpuRamp := ui.NewColorRamp(0.0, 1.0, colorLoadBarMinCPU, colorLoadBarMaxCPU)

	const text = " Legend: CPU RAM "
	const barsOffset = 9
	legendX := rightmostFrameBorder - len(text) // Leave some space to the right
	drawText(screen, legendX, row, text, twin.StyleDefault.WithForeground(colorText))

	cpuLoadBar := ui.NewLoadBar(legendX+barsOffset, legendX+3+barsOffset, cpuRamp)
	cpuLoadBar.SetCellBackground(screen, legendX+barsOffset, row, 1.0)
	cpuLoadBar.SetCellBackground(screen, legendX+barsOffset+1, row, 1.0)
	cpuLoadBar.SetCellBackground(screen, legendX+barsOffset+2, row, 1.0)

	memLoadBar := ui.NewLoadBar(legendX+barsOffset+4, legendX+barsOffset+6, memoryRamp)
	memLoadBar.SetCellBackground(screen, legendX+barsOffset+4, row, 1.0)
	memLoadBar.SetCellBackground(screen, legendX+barsOffset+5, row, 1.0)
	memLoadBar.SetCellBackground(screen, legendX+barsOffset+6, row, 1.0)
}
