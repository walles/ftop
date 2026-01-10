package ptop

import (
	"fmt"
	"strconv"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/processes"
	"github.com/walles/ptop/internal/ui"
)

// Render three tables and combine them: per-process (on the left), per-user
// (top right), and per-command (bottom right).
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
	[]commandStats,
) {
	usersHeight := processesHeight/2 - 1
	commandsHeight := processesHeight - usersHeight

	procsHeaders := []string{
		"PID", "Command", "Username", "CPU", "Time", "RAM",
	}
	usersHeaders := []string{
		"Username", "CPU", "RAM",
	}
	commandsHeaders := []string{
		"Command", "CPU", "RAM",
	}

	procsTable := [][]string{
		procsHeaders,
	}
	processesByScore := SortByScore(processesRaw, func(p processes.Process) stats {
		return stats{
			// The name in this case is really the third sort key. Since PIDs
			// are unique, we want to use those for sorting if RAM and CPU time
			// are equal.
			name:    strconv.Itoa(p.Pid),
			cpuTime: p.CpuTimeOrZero(),
			rssKb:   p.RssKb,
		}
	})
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

	users := aggregate(processesRaw, func(p processes.Process) string { return p.Username }, func(stat stats) userStats {
		return userStats{stats: stat}
	})
	users = SortByScore(users, func(u userStats) stats {
		return u.stats
	})

	usersTable := [][]string{
		usersHeaders,
	}
	for _, u := range users {
		if len(usersTable) >= usersHeight {
			break
		}

		row := []string{
			u.name,
			ui.FormatDuration(u.cpuTime),
			ui.FormatMemory(1024 * int64(u.rssKb)),
		}

		usersTable = append(usersTable, row)
	}
	for len(usersTable) < usersHeight {
		usersTable = append(usersTable, make([]string, len(usersHeaders)))
	}

	commands := aggregate(processesRaw, func(p processes.Process) string { return p.Command }, func(stat stats) commandStats {
		return commandStats{stats: stat}
	})
	commands = SortByScore(commands, func(b commandStats) stats {
		return b.stats
	})

	commandsTable := [][]string{
		commandsHeaders,
	}
	for _, b := range commands {
		if len(commandsTable) >= commandsHeight {
			break
		}

		row := []string{
			b.name,
			ui.FormatDuration(b.cpuTime),
			ui.FormatMemory(1024 * int64(b.rssKb)),
		}

		commandsTable = append(commandsTable, row)
	}
	for len(commandsTable) < commandsHeight {
		commandsTable = append(commandsTable, make([]string, len(commandsHeaders)))
	}

	combinedTable := [][]string{}

	// If the users table would be 1 long:
	// 0: users header
	// 1: --- bottom separator ---
	// 2: --- top separator ---
	// 3: commands start here
	//
	// So the commands start at 1 + 2 = 3
	commandsStartRow := len(usersTable) + 2
	for i, procRow := range procsTable {
		row := make([]string, 0, len(procRow)+len(usersTable[0]))
		row = append(row, procRow...)
		if i < len(usersTable) {
			row = append(row, usersTable[i]...)
		} else if i >= commandsStartRow {
			commandsIndex := i - commandsStartRow
			row = append(row, commandsTable[commandsIndex]...)
		} else {
			// Neither user nor command row, pad with empty cells
			for range commandsHeaders {
				row = append(row, "")
			}
		}

		combinedTable = append(combinedTable, row)
	}

	return combinedTable, len(usersTable), processesByScore, users, commands
}

// Render the three sections: per-process (on the left), per-user (top right),
// and per-command (bottom right).
//
// The processes table contains cells for all three sections: per-process (on
// the left), per-user (top right), and per-command (bottom right).
//
// topRow and bottomRow are screen rows. Screen borders go on those rows.
//
// usersHeight is the number of table lines in the per-user section, including
// borders. Borders is not included in this number. The commands table will use
// the remaining space below the users table.
func renderProcessesBlock(
	screen twin.Screen,
	table [][]string,
	processes []processes.Process,
	firstScreenRow int,
	bottomRow int,
	users []userStats,
	usersHeight int,
	commands []commandStats,
) {
	width, _ := screen.Size()

	// -2 for borders, -5 for column dividers, -2 for the two borders between
	// sections and -2 for column dividers in the right section
	availableToColumns := width - 2 - 5 - 2 - 2

	// Don't grow the PID column, that looks weird
	widths := ui.ColumnWidths(table, availableToColumns, false)

	perProcessTableWidth := widths[0] + 1 + widths[1] + 1 + widths[2] + 1 + widths[3] + 1 + widths[4] + 1 + widths[5]
	rightPerProcessBorderColumn := perProcessTableWidth + 1    // Screen column. +1 for the left frame line.
	leftPerUserBorderColumn := rightPerProcessBorderColumn + 1 // Screen column

	usersBottomBorder := firstScreenRow + 1 + usersHeight
	commandsTopRow := usersBottomBorder + 1

	renderProcesses(screen, 0, firstScreenRow, rightPerProcessBorderColumn, bottomRow, table, widths, processes)
	renderPerUser(screen, leftPerUserBorderColumn, firstScreenRow, width-1, usersBottomBorder, table, widths, users)

	// Skip the per-user rows. If usersHeight is 0:
	// 0: post-users separator line
	// 1: post-users separator line number two
	// 2: commands start here
	//
	// So for usersHeight = 0, we should start at index 2
	table = table[usersHeight+2:]
	renderPerCommand(screen, leftPerUserBorderColumn, commandsTopRow, width-1, bottomRow, table, widths, commands)
}

func renderProcesses(screen twin.Screen, x0, y0, x1, y1 int, table [][]string, widths []int, processes []processes.Process) {
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
	// +2 = ignore top border and the header line
	topBottomRamp := ui.NewColorRamp(float64(y0+2), float64(y1-1), colorTop, colorBottom)

	userColumn0 := x0 + 1 + widths[0] + 1 + widths[1] // Screen column
	userColumnN := userColumn0 + widths[2] - 1        // Screen column
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

	perProcessCpuAndMemBar := ui.NewOverlappingLoadBars(x0+1, x1-1, cpuRamp, memoryRamp)

	//
	// Render table contents
	//

	for rowIndex, row := range table {
		line := fmt.Sprintf(formatString,
			row[0], row[1], row[2], row[3], row[4], row[5],
		)

		y := y0 + 1 + rowIndex // screen row

		var rowStyle twin.Style
		if rowIndex == 0 {
			// Header row, header style
			rowStyle = twin.StyleDefault.WithAttr(twin.AttrBold)
		} else {
			rowStyle = twin.StyleDefault
			rowStyle = rowStyle.WithForeground(topBottomRamp.AtInt(y))
		}

		x := x0 + 1 // screen column
		for _, char := range line {
			style := rowStyle
			if x >= userColumn0 && x <= userColumnN {
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

			x++
		}
	}

	renderFrame(screen, x0, y0, x1, y1, "By Process")
	renderLegend(screen, y1, x1)
}

func renderPerUser(screen twin.Screen, x0, y0, x1, y1 int, table [][]string, widths []int, users []userStats) {
	widths = widths[6:] // Skip the per-process columns

	// Formats are "%5.5s" or "%-5.5s", where "5.5" means "pad and truncate to
	// 5", and the "-" means left-align.
	formatString := fmt.Sprintf("%%-%d.%ds %%%d.%ds %%%d.%ds",
		widths[0], widths[0],
		widths[1], widths[1],
		widths[2], widths[2],
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
	// +2 = ignore top border and the header line
	topBottomRamp := ui.NewColorRamp(float64(y0+2), float64(y1-1), colorTop, colorBottom)

	usernameColumn0 := x0 + 1                          // Screen column
	usernameColumnN := usernameColumn0 + widths[0] - 1 // Screen column
	currentUsername := getCurrentUsername()

	// If y0 = 0 and y1 = 1, then there would be 0 content rows between the
	// borders
	rowsWithHeaderWithoutBorders := y1 - y0 - 1

	maxCpuSecondsPerUser := 0.0
	maxRssKbPerUser := 0
	for _, u := range users {
		if u.cpuTime.Seconds() > maxCpuSecondsPerUser {
			maxCpuSecondsPerUser = u.cpuTime.Seconds()
		}
		if u.rssKb > maxRssKbPerUser {
			maxRssKbPerUser = u.rssKb
		}
	}

	cpuAndMemBar := ui.NewOverlappingLoadBars(x0+1, x1-1, cpuRamp, memoryRamp)

	//
	// Render table contents
	//

	for rowIndex, row := range table {
		if rowIndex >= rowsWithHeaderWithoutBorders {
			// No more room
			break
		}

		row = row[6:] // Skip the per-process columns
		line := fmt.Sprintf(formatString,
			row[0], row[1], row[2],
		)

		y := y0 + 1 + rowIndex // screen row

		var rowStyle twin.Style
		if rowIndex == 0 {
			// Header row, header style
			rowStyle = twin.StyleDefault.WithAttr(twin.AttrBold)
		} else {
			rowStyle = twin.StyleDefault
			rowStyle = rowStyle.WithForeground(topBottomRamp.AtInt(y))
		}

		x := x0 + 1 // screen column
		for _, char := range line {
			style := rowStyle
			if x >= usernameColumn0 && x <= usernameColumnN {
				username := row[0]
				if username == currentUsername {
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
				cpuAndMemBar.SetCellBackground(screen, x, y, cpuFraction, memFraction)
			}

			x++
		}
	}

	renderFrame(screen, x0, y0, x1, y1, "By User")
}

// Assumes the first row of the table contains the commands header line
func renderPerCommand(screen twin.Screen, x0, y0, x1, y1 int, table [][]string, widths []int, commands []commandStats) {
	widths = widths[6:] // Skip the per-process columns

	// Formats are "%5.5s" or "%-5.5s", where "5.5" means "pad and truncate to
	// 5", and the "-" means left-align.
	formatString := fmt.Sprintf("%%-%d.%ds %%%d.%ds %%%d.%ds",
		widths[0], widths[0],
		widths[1], widths[1],
		widths[2], widths[2],
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
	// +2 = ignore top border and the header line
	topBottomRamp := ui.NewColorRamp(float64(y0+2), float64(y1-1), colorTop, colorBottom)

	// If y0 = 0 and y1 = 1, then there would be 0 content rows between the
	// borders
	rowsWithHeaderWithoutBorders := y1 - y0 - 1

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
		if rowIndex >= rowsWithHeaderWithoutBorders {
			// No more room
			break
		}

		row = row[6:] // Skip the per-process columns
		line := fmt.Sprintf(formatString,
			row[0], row[1], row[2],
		)

		y := y0 + 1 + rowIndex // screen row

		var rowStyle twin.Style
		if rowIndex == 0 {
			// Header row, header style
			rowStyle = twin.StyleDefault.WithAttr(twin.AttrBold)
		} else {
			rowStyle = twin.StyleDefault
			rowStyle = rowStyle.WithForeground(topBottomRamp.AtInt(y))
		}

		x := x0 + 1 // screen column
		for _, char := range line {
			style := rowStyle
			screen.SetCell(x, y, twin.StyledRune{Rune: char, Style: style})

			if rowIndex == 0 {
				// Header row, no load bars here
				x++
				continue
			}

			index := rowIndex - 1 // Because rowIndex 0 is the header
			if index < len(commands) {
				command := commands[index]
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

	renderFrame(screen, x0, y0, x1, y1, "By Command")
}

// Towards the right, draw "CPU" with a CPU load bar behind it, and "RAM" with a
// RAM load bar behind it.
func renderLegend(screen twin.Screen, y int, rightFrameBorder int) {
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
	legendX := rightFrameBorder - len(text) // Leave some space to the right
	drawText(screen, legendX, y, text, twin.StyleDefault.WithForeground(colorText))

	cpuLoadBar := ui.NewLoadBar(legendX+barsOffset, legendX+3+barsOffset, cpuRamp)
	cpuLoadBar.SetCellBackground(screen, legendX+barsOffset, y, 1.0)
	cpuLoadBar.SetCellBackground(screen, legendX+barsOffset+1, y, 1.0)
	cpuLoadBar.SetCellBackground(screen, legendX+barsOffset+2, y, 1.0)

	memLoadBar := ui.NewLoadBar(legendX+barsOffset+4, legendX+barsOffset+6, memoryRamp)
	memLoadBar.SetCellBackground(screen, legendX+barsOffset+4, y, 1.0)
	memLoadBar.SetCellBackground(screen, legendX+barsOffset+5, y, 1.0)
	memLoadBar.SetCellBackground(screen, legendX+barsOffset+6, y, 1.0)
}
