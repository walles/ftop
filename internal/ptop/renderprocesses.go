package ptop

import (
	"fmt"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/processes"
	"github.com/walles/ptop/internal/ui"
)

// Render three tables and combine them: per-process (on the left), per-user
// (top right), and per-binary (bottom right).
//
// Returns the combined table, as well as the row count (including headers) of
// the per-user section.
//
// processesHeight is the height of the table, without borders
func createProcessesTable(processesRaw []processes.Process, processesHeight int) (
	[][]string,
	int,
	[]processes.Process,
	[]userStats,
	[]binaryStats,
) {
	usersHeight := processesHeight / 2
	binariesHeight := processesHeight - usersHeight

	procsHeaders := []string{
		"PID", "Command", "Username", "CPU", "Time", "RAM",
	}
	usersHeaders := []string{
		"Username", "CPU", "RAM",
	}
	binariesHeaders := []string{
		"Binary", "CPU", "RAM",
	}

	procsTable := [][]string{
		procsHeaders,
	}
	processesByScore := ProcessesByScore(processesRaw)
	for _, p := range processesByScore {
		if len(procsTable) >= processesHeight {
			break
		}

		row := []string{
			fmt.Sprintf("%d", p.Pid),
			p.Command,
			p.Username,
			p.CpuPercentString(),
			p.CpuTimeString(),
			ui.FormatMemory(int64(p.RssKb) * 1024),
		}

		procsTable = append(procsTable, row)
	}
	for len(procsTable) < processesHeight {
		procsTable = append(procsTable, make([]string, len(procsHeaders)))
	}

	usersTable := [][]string{
		usersHeaders,
	}
	users := UsersByScore(processesRaw)
	for _, u := range users {
		if len(usersTable) >= usersHeight {
			break
		}

		row := []string{
			u.username,
			ui.FormatDuration(u.cpuTime),
			ui.FormatMemory(1024 * int64(u.rssKb)),
		}

		usersTable = append(usersTable, row)
	}
	for len(usersTable) < usersHeight {
		usersTable = append(usersTable, make([]string, len(usersHeaders)))
	}

	binariesTable := [][]string{
		binariesHeaders,
	}
	binaries := BinariesByScore(processesRaw)
	for _, b := range binaries {
		if len(binariesTable) >= binariesHeight {
			break
		}

		row := []string{
			b.binaryName,
			ui.FormatDuration(b.cpuTime),
			ui.FormatMemory(1024 * int64(b.rssKb)),
		}

		binariesTable = append(binariesTable, row)
	}
	for len(binariesTable) < binariesHeight {
		binariesTable = append(binariesTable, make([]string, len(binariesHeaders)))
	}

	combinedTable := [][]string{}

	// If the users table would be 1 long:
	// 0: users header
	// 1: --- bottom separator ---
	// 2: --- top separator ---
	// 3: binaries start here
	//
	// So the binaries start at 1 + 2 = 3
	binariesStartRow := len(usersTable) + 2
	for i, procRow := range procsTable {
		row := make([]string, 0, len(procRow)+len(usersTable[0]))
		row = append(row, procRow...)
		if i < len(usersTable) {
			row = append(row, usersTable[i]...)
		} else if i >= binariesStartRow {
			binariesIndex := i - binariesStartRow
			row = append(row, binariesTable[binariesIndex]...)
		} else {
			// Neither user nor binary row, pad with empty cells
			for range binariesHeaders {
				row = append(row, "")
			}
		}

		combinedTable = append(combinedTable, row)
	}

	return combinedTable, len(usersTable), processesByScore, users, binaries
}

// The processes table contains cells for all three sections: per-process (on
// the left), per-user (top right), and per-binary (bottom right).
//
// topRow and bottomRow are screen rows. Screen borders go on those rows.
//
// usersHeight is the number of table lines in the per-user section, including
// borders. Borders is not included in this number. The binaries table will use
// the remaining space below the users table.
func renderProcesses(
	screen twin.Screen,
	table [][]string,
	processes []processes.Process,
	firstScreenRow int,
	bottomRow int,
	users []userStats,
	usersHeight int,
	binaries []binaryStats,
) {
	width, _ := screen.Size()

	// Don't grow the PID column, that looks weird
	widths := ui.ColumnWidths(table, width-2, false)

	perProcessTableWidth := widths[0] + 1 + widths[1] + 1 + widths[2] + 1 + widths[3] + 1 + widths[4] + 1 + widths[5]
	rightPerProcessBorderColumn := perProcessTableWidth + 1 // +1 for the left frame line
	leftPerUserBorderColumn := rightPerProcessBorderColumn + 1
	rightPerUserBorderColumn := leftPerUserBorderColumn + widths[6] + 1 + widths[7] + 1 + widths[8] + 1

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

	//
	// Rendering setup
	//

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

	perUserTableWidth := widths[6] + 1 + widths[7] + 1 + widths[8]
	perUserTableScreenColumn := perProcessTableWidth + 2 // +2 for the "||" divider

	processUserColumn0 := 1 + widths[0] + 1 + widths[1] + 1
	processUserColumnN := processUserColumn0 + widths[2] - 1
	userUserColumn0 := 1 + perUserTableScreenColumn
	userUserColumnN := userUserColumn0 + widths[6] - 1
	currentUsername := getCurrentUsername()

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

	maxCpuSecondsPerBinary := 0.0
	maxRssKbPerBinary := 0
	for _, b := range binaries {
		cpuSeconds := b.cpuTime.Seconds()
		if cpuSeconds > maxCpuSecondsPerBinary {
			maxCpuSecondsPerBinary = cpuSeconds
		}
		if b.rssKb > maxRssKbPerBinary {
			maxRssKbPerBinary = b.rssKb
		}
	}

	perProcessCpuAndMemBar := ui.NewOverlappingLoadBars(
		1, 1+perProcessTableWidth-1, cpuRamp, memoryRamp)
	perUserCpuAndMemBar := ui.NewOverlappingLoadBars(
		1+perUserTableScreenColumn, 1+perUserTableScreenColumn+perUserTableWidth-1, cpuRamp, memoryRamp)

	//
	// Render table contents
	//

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
		x := 1

		// y is a screen row
		y := firstScreenRow + 1 + rowIndex

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

			screen.SetCell(x, y, twin.StyledRune{Rune: char, Style: style})

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
				perProcessCpuAndMemBar.SetCellBackground(screen, x, y, cpuFraction, memFraction)
			}

			if index < usersHeight-1 {
				user := users[index]
				cpuFraction := 0.0
				if maxCpuSecondsPerUser > 0.0 {
					cpuFraction = user.cpuTime.Seconds() / maxCpuSecondsPerUser
				}
				memFraction := 0.0
				if maxRssKbPerUser > 0 {
					memFraction = float64(user.rssKb) / float64(maxRssKbPerUser)
				}
				perUserCpuAndMemBar.SetCellBackground(screen, x, y, cpuFraction, memFraction)

				// Bold the current username
				if user.username == currentUsername && x >= userUserColumn0 && x <= userUserColumnN {
					cell := screen.GetCell(x, y)
					cell.Style = cell.Style.WithAttr(twin.AttrBold)
					screen.SetCell(x, y, cell)
				}
			}

			if index == usersHeight-1 {
				// Bold the binaries header row
				if x >= userUserColumn0 {
					cell := screen.GetCell(x, y)
					cell.Style = cell.Style.WithAttr(twin.AttrBold)
					screen.SetCell(x, y, cell)
				}
			}

			x++
		}
	}

	//
	// Render frames
	//

	renderFrame(
		screen,
		firstScreenRow,
		0,
		bottomRow,
		rightPerProcessBorderColumn,
		"By process",
	)
	renderLegend(screen, bottomRow, rightPerProcessBorderColumn)

	usersBottomBorder := firstScreenRow + 1 + usersHeight - 2 // -2 to skip the borders
	renderFrame(
		screen,
		firstScreenRow,
		leftPerUserBorderColumn,
		usersBottomBorder,
		rightPerUserBorderColumn,
		"By user",
	)

	binariesTopRow := usersBottomBorder + 1
	binariesBottomRow := bottomRow
	renderFrame(
		screen,
		binariesTopRow,
		leftPerUserBorderColumn,
		binariesBottomRow,
		rightPerUserBorderColumn,
		"By binary",
	)
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
