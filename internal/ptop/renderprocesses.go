package ptop

import (
	"fmt"
	"strconv"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/processes"
	"github.com/walles/ptop/internal/themes"
	"github.com/walles/ptop/internal/ui"
)

// Render the three sections: per-process (on the left), per-user (top right),
// and per-command (bottom right).
//
// y0 and y1 are screen rows and are both inclusive. Borders will be drawn on
// those rows.
//
// Returns true if the screen was wide enough, otherwise false.
func tryRenderThreeProcessPanes(screen twin.Screen, theme themes.Theme, processesRaw []processes.Process, y0 int, y1 int) bool {
	// Including borders. If they are the same, the height is still 1.
	renderHeight := y1 - y0 + 1

	// -2 for borders, they won't be part of the table
	table, usersHeight, processes, users, commands := createProcessesTable(processesRaw, renderHeight-2)

	width, _ := screen.Size()

	// -2 for borders, -5 for column dividers, -2 for the two borders between
	// sections and -2 for column dividers in the right section
	availableToColumns := width - 2 - 5 - 2 - 2

	// Don't grow the PID column, that looks weird
	widths := ui.ColumnWidths(table, availableToColumns, false)

	// Check that all CPU values fit. The header is not required to fit.
	cpuColumnIndex := 3
	cpuColumnWidth := widths[cpuColumnIndex]
	for rowIndex, row := range table {
		if rowIndex == 0 {
			// Header row
			continue
		}

		cpuValue := row[cpuColumnIndex]
		if len(cpuValue) > cpuColumnWidth {
			return false
		}
	}

	perProcessTableWidth := widths[0] + 1 + widths[1] + 1 + widths[2] + 1 + widths[3] + 1 + widths[4] + 1 + widths[5]
	rightPerProcessBorderColumn := perProcessTableWidth + 1    // Screen column. +1 for the left frame line.
	leftPerUserBorderColumn := rightPerProcessBorderColumn + 1 // Screen column

	usersBottomBorder := y0 + 1 + usersHeight
	commandsTopRow := usersBottomBorder + 1

	renderProcesses(screen, theme, 0, y0, rightPerProcessBorderColumn, y1, table, widths, processes)
	renderPerUser(screen, theme, leftPerUserBorderColumn, y0, width-1, usersBottomBorder, table, widths, users)

	// Skip the per-user rows. If usersHeight is 0:
	// 0: post-users separator line
	// 1: post-users separator line number two
	// 2: commands start here
	//
	// So for usersHeight = 0, we should start at index 2
	table = table[usersHeight+2:]
	renderPerCommand(screen, theme, leftPerUserBorderColumn, commandsTopRow, width-1, y1, table, widths, commands)

	return true
}

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

	usersTable := [][]string{}
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
		usersTable = append(usersTable, make([]string, 3))
	}

	commands := aggregate(processesRaw, func(p processes.Process) string { return p.Command }, func(stat stats) commandStats {
		return commandStats{stats: stat}
	})
	commands = SortByScore(commands, func(b commandStats) stats {
		return b.stats
	})

	commandsTable := [][]string{}
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
		commandsTable = append(commandsTable, make([]string, 3))
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
			for range 3 {
				row = append(row, "")
			}
		}

		combinedTable = append(combinedTable, row)
	}

	return combinedTable, len(usersTable), processesByScore, users, commands
}

func renderProcesses(screen twin.Screen, theme themes.Theme, x0, y0, x1, y1 int, table [][]string, widths []int, processes []processes.Process) {
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

	memoryRamp := ui.NewColorRamp(0.0, 1.0, theme.LoadBarMin(), theme.LoadBarMaxRam())
	cpuRamp := ui.NewColorRamp(0.0, 1.0, theme.LoadBarMin(), theme.LoadBarMaxCpu())

	// +2 = ignore top border and the header line
	topBottomRamp := ui.NewColorRamp(float64(y0+2), float64(y1-1), theme.Foreground(), theme.FadedForeground())

	userColumn0 := x0 + 1 + widths[0] + 1 + widths[1] // Screen column
	userColumnN := userColumn0 + widths[2] - 1        // Screen column
	currentUsername := getCurrentUsername()

	commandColumn0 := x0 + widths[0] + 1             // Screen column
	commandColumnN := commandColumn0 + widths[1] - 1 // Screen column

	// +2 = ignore top border and the header line
	userRamp := ui.NewColorRamp(float64(y0+2), float64(y1-1), theme.HighlightedForeground(), theme.FadedForeground())

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
			if rowIndex > 0 && x >= commandColumn0 && x <= commandColumnN {
				style = style.WithForeground(userRamp.AtInt(y))
			}

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

	renderFrame(screen, theme, x0, y0, x1, y1, "By Process")
	renderLegend(screen, theme, y1, x1)
}

// Towards the right, draw "CPU" with a CPU load bar behind it, and "RAM" with a
// RAM load bar behind it.
func renderLegend(screen twin.Screen, theme themes.Theme, y int, rightFrameBorder int) {
	// Turn up the bottom color this much so it's visible in the small legend
	const adjustUp = 0.5

	colorLoadBarMaxRAM := theme.LoadBarMaxRam()
	colorLoadBarMinRAM := theme.LoadBarMin().Mix(colorLoadBarMaxRAM, adjustUp)

	colorLoadBarMaxCPU := theme.LoadBarMaxCpu()
	colorLoadBarMinCPU := theme.LoadBarMin().Mix(colorLoadBarMaxCPU, adjustUp)

	memoryRamp := ui.NewColorRamp(0.0, 1.0, colorLoadBarMinRAM, colorLoadBarMaxRAM)
	cpuRamp := ui.NewColorRamp(0.0, 1.0, colorLoadBarMinCPU, colorLoadBarMaxCPU)

	const textLegend = " Legend:"
	const textCpuRam = " CPU RAM "
	const barsOffset = 9
	legendX := rightFrameBorder - len(textLegend) - len(textCpuRam)
	x := legendX
	x += drawText(
		screen,
		x,
		y,
		rightFrameBorder,
		textLegend,
		twin.StyleDefault.WithForeground(theme.Border()),
	)
	drawText(
		screen,
		x,
		y,
		rightFrameBorder,
		textCpuRam,
		twin.StyleDefault.WithForeground(theme.Foreground()),
	)

	cpuLoadBar := ui.NewLoadBar(legendX+barsOffset, legendX+3+barsOffset, cpuRamp)
	cpuLoadBar.SetCellBackground(screen, legendX+barsOffset, y, 1.0)
	cpuLoadBar.SetCellBackground(screen, legendX+barsOffset+1, y, 1.0)
	cpuLoadBar.SetCellBackground(screen, legendX+barsOffset+2, y, 1.0)

	memLoadBar := ui.NewLoadBar(legendX+barsOffset+4, legendX+barsOffset+6, memoryRamp)
	memLoadBar.SetCellBackground(screen, legendX+barsOffset+4, y, 1.0)
	memLoadBar.SetCellBackground(screen, legendX+barsOffset+5, y, 1.0)
	memLoadBar.SetCellBackground(screen, legendX+barsOffset+6, y, 1.0)
}
