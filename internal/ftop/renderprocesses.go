package ftop

import (
	"fmt"

	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/ftop/internal/ui"
	"github.com/walles/ftop/internal/util"
	"github.com/walles/moor/v2/twin"
)

func (u *Ui) canRenderThreeProcessPanes(screen twin.Screen, processesRaw []processes.Process, y0 int, y1 int) bool {
	// Including borders. If they are the same, the height is still 1.
	renderHeight := y1 - y0 + 1

	// -2 for borders, they won't be part of the table
	table, _, _, _, _ := u.createProcessesTable(processesRaw, renderHeight-2)

	width, _ := screen.Size()

	// -2 for borders, -5 for column dividers, -2 for the two borders between
	// sections and -2 for column dividers in the right section
	availableToColumns := width - 2 - 5 - 2 - 2

	// Don't grow the PID column, that looks weird
	widths := ui.ColumnWidths(table, availableToColumns, false)

	return isWideEnough(table, widths)
}

// Render the three sections: per-process (on the left), per-user (top right),
// and per-command (bottom right).
//
// y0 and y1 are screen rows and are both inclusive. Borders will be drawn on
// those rows.
func (u *Ui) renderThreeProcessPanes(processesRaw []processes.Process, y0 int, y1 int) {
	// Including borders. If they are the same, the height is still 1.
	renderHeight := y1 - y0 + 1

	// -2 for borders, they won't be part of the table
	table, usersHeight, processes, users, commands := u.createProcessesTable(processesRaw, renderHeight-2)

	width, _ := u.screen.Size()

	// -2 for borders, -5 for column dividers, -2 for the two borders between
	// sections and -2 for column dividers in the right section
	availableToColumns := width - 2 - 5 - 2 - 2

	// Don't grow the PID column, that looks weird
	widths := ui.ColumnWidths(table, availableToColumns, false)

	perProcessTableWidth := widths[0] + 1 + widths[1] + 1 + widths[2] + 1 + widths[3] + 1 + widths[4] + 1 + widths[5]
	rightPerProcessBorderColumn := perProcessTableWidth + 1    // Screen column. +1 for the left frame line.
	leftPerUserBorderColumn := rightPerProcessBorderColumn + 1 // Screen column

	usersBottomBorder := y0 + 1 + usersHeight
	commandsTopRow := usersBottomBorder + 1

	u.renderProcesses(0, y0, rightPerProcessBorderColumn, y1, table, widths, processes)
	renderPerUser(u.screen, u.theme, leftPerUserBorderColumn, y0, width-1, usersBottomBorder, table, widths, users)

	// Skip the per-user rows. If usersHeight is 0:
	// 0: post-users separator line
	// 1: post-users separator line number two
	// 2: commands start here
	//
	// So for usersHeight = 0, we should start at index 2
	table = table[usersHeight+2:]
	renderPerCommand(u.screen, u.theme, leftPerUserBorderColumn, commandsTopRow, width-1, y1, table, widths, commands)
}

func isWideEnough(table [][]string, widths []int) bool {
	columnsThatMustFit := []int{
		3, // CPU
		4, // Time
		5, // RAM
		7, // User / Command Time
		8, // User / Command RAM
	}

	for rowIndex, row := range table {
		for _, colIndex := range columnsThatMustFit {
			if rowIndex == 0 && colIndex < 6 {
				// Header row, doesn't need to fit
				continue
			}

			cellValue := row[colIndex]
			if len(cellValue) > widths[colIndex] {
				return false
			}
		}
	}

	return true
}

func (u *Ui) renderSingleProcessesPane(processesRaw []processes.Process, y0 int, y1 int) {
	// Including borders. If they are the same, the height is still 1.
	renderHeight := y1 - y0 + 1

	// -2 for borders, they won't be part of the table
	table, _, processes, _, _ := u.createProcessesTable(processesRaw, renderHeight-2)

	// Drop the three rightmost columns (per-user and per-command) from the
	// table
	for rowIndex, row := range table {
		table[rowIndex] = row[:6]
	}

	width, _ := u.screen.Size()

	// -2 for borders, -5 for column dividers
	availableToColumns := width - 2 - 5

	// Don't grow the PID column, that looks weird
	widths := ui.ColumnWidths(table, availableToColumns, false)

	u.renderProcesses(0, y0, width-1, y1, table, widths, processes)
}

// Render three tables and combine them: per-process (on the left), per-user
// (top right), and per-command (bottom right).
//
// Returns the combined table, as well as the row count (including headers) of
// the per-user section.
//
// processesHeight is the height of the table, without borders
func (u *Ui) createProcessesTable(processesRaw []processes.Process, processesHeight int) (
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
	processesByScore := sortProcessesForDisplay(processesRaw)

	processesByScore = u.fixPickedProcess(processesByScore)

	for _, p := range processesByScore {
		if len(procsTable) >= processesHeight {
			break
		}

		row := []string{
			fmt.Sprintf("%d", p.Pid),
			p.Command + p.DeduplicationSuffix,
			p.Username,
			p.CpuPercentString(),
			p.CpuTimeString(),
			util.FormatMemory(int64(p.RssKb) * 1024),
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
			util.FormatDuration(u.cpuTime),
			util.FormatMemory(1024 * int64(u.rssKb)),
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
			util.FormatDuration(b.cpuTime),
			util.FormatMemory(1024 * int64(b.rssKb)),
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

// Will provide cells covering at least width screen columns
func renderCommand(command string, deduplicationSuffix string, width int, textColor twin.Color) []twin.StyledRune {
	result := make([]twin.StyledRune, 0, width)
	resultWidth := 0 // In screen columns

	// Draw the command
	for _, char := range command {
		styledRune := twin.StyledRune{Rune: char, Style: twin.StyleDefault.WithForeground(textColor)}
		result = append(result, styledRune)
		resultWidth += styledRune.Width()
	}

	// Draw the deduplication suffix (faint)
	for _, char := range deduplicationSuffix {
		styledRune := twin.StyledRune{
			Rune:  char,
			Style: twin.StyleDefault.WithForeground(textColor).WithAttr(twin.AttrDim),
		}

		result = append(result, styledRune)
		resultWidth += styledRune.Width()
	}

	for resultWidth < width {
		result = append(result, twin.StyledRune{Rune: ' ', Style: twin.StyleDefault.WithForeground(textColor)})
		resultWidth++
	}

	return result
}

func (u *Ui) renderProcesses(x0, y0, x1, y1 int, table [][]string, widths []int, procs []processes.Process) {
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

	memoryRamp := ui.NewColorRamp(0.0, 1.0, u.theme.LoadBarMin(), u.theme.LoadBarMaxRam())
	cpuRamp := ui.NewColorRamp(0.0, 1.0, u.theme.LoadBarMin(), u.theme.LoadBarMaxCpu())

	// +2 = ignore top border and the header line
	topBottomRamp := ui.NewColorRamp(float64(y0+2), float64(y1-1), u.theme.Foreground(), u.theme.FadedForeground())

	userColumn0 := x0 + 1 + widths[0] + 1 + widths[1] // Screen column
	userColumnN := userColumn0 + widths[2]            // Screen column
	currentUsername := util.GetCurrentUsername()

	commandColumn0 := x0 + 1 + widths[0] + 1         // Screen column. x0 + 1 for left border, then PID column, then a space separator
	commandColumnN := commandColumn0 + widths[1] - 1 // Screen column

	// +2 = ignore top border and the header line
	userRamp := ui.NewColorRamp(float64(y0+2), float64(y1-1), u.theme.HighlightedForeground(), u.theme.FadedForeground())

	maxCpuSecondsPerProcess := 0.0
	maxRssKbPerProcess := 0
	for _, p := range procs {
		if p.CpuTime != nil && p.CpuTime.Seconds() > maxCpuSecondsPerProcess {
			maxCpuSecondsPerProcess = p.CpuTime.Seconds()
		}
		if p.RssKb > maxRssKbPerProcess {
			maxRssKbPerProcess = p.RssKb
		}
	}

	// Pretend the load bar starts at x0, even though it really starts at x0+1.
	// This way, even the leftmost cell (at x0+1) will get a non-background
	// color. The effect is especially visible for processes with low PIDs and
	// short load bars, like the init (PID 1) process.
	perProcessCpuAndMemBar := ui.NewOverlappingLoadBars(x0, x1-1, cpuRamp, memoryRamp)

	//
	// Render table contents
	//

	for rowIndex, row := range table {
		line := fmt.Sprintf(formatString,
			row[0], row[1], row[2], row[3], row[4], row[5],
		)

		var process *processes.Process
		if rowIndex > 0 && rowIndex-1 < len(procs) {
			process = &procs[rowIndex-1]
		}

		y := y0 + 1 + rowIndex // screen row

		var commandCells []twin.StyledRune
		if process != nil {
			commandCells = renderCommand(process.Command, process.DeduplicationSuffix, widths[1], userRamp.AtInt(y))
		} else {
			// Cover the command column with empty cells
			commandCells = make([]twin.StyledRune, 0, widths[1])
			space := twin.StyledRune{Rune: ' ', Style: twin.StyleDefault.WithForeground(userRamp.AtInt(y))}
			for len(commandCells) < widths[1] {
				commandCells = append(commandCells, space)
			}
		}

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
			char := twin.StyledRune{Rune: char, Style: rowStyle}

			if rowIndex > 0 && x >= commandColumn0 && x <= commandColumnN {
				// Command column

				// FIXME: Given some cells are multiple screen columns wide, will this work?
				char = commandCells[x-commandColumn0]
			} else if rowIndex > 0 && x >= userColumn0 && x <= userColumnN {
				// User column
				username := row[2]
				if username == "root" && currentUsername != "root" {
					char.Style = char.Style.WithAttr(twin.AttrDim)
				} else if username != currentUsername {
					char.Style = char.Style.WithAttr(twin.AttrBold)
				}
			}

			if u.pickedLine != nil && *u.pickedLine == rowIndex-1 {
				// Picked process line, highlight it!
				char.Style = twin.StyleDefault.WithAttr(twin.AttrReverse)
			}

			u.screen.SetCell(x, y, char)

			if u.pickedLine != nil && *u.pickedLine == rowIndex-1 {
				// Picked process line, don't draw any load bars since they will
				// mess up the highlighting.
				x += char.Width()
				continue
			}

			if rowIndex == 0 {
				// Header row, no load bars here
				x += char.Width()
				continue
			}

			if process != nil {
				cpuFraction := 0.0
				if process.CpuTime != nil && maxCpuSecondsPerProcess > 0.0 {
					cpuFraction = process.CpuTime.Seconds() / maxCpuSecondsPerProcess
				}
				memFraction := 0.0
				if maxRssKbPerProcess > 0 {
					memFraction = float64(process.RssKb) / float64(maxRssKbPerProcess)
				}
				perProcessCpuAndMemBar.SetCellBackground(u.screen, x, y, cpuFraction, memFraction)
			}

			x += char.Width()
		}
	}

	const byProcess = "By Process"
	renderFrame(u.screen, u.theme, x0, y0, x1, y1, byProcess)

	pickUpArrow := u.pickedLine != nil

	// Down arrow is available if we can move down, limited by both screen height and process count
	lastVisibleProcessIndex := len(table) - 2 // -1 for header, -1 for zero-indexing
	lastProcessIndex := len(procs) - 1
	maxPickableIndex := min(lastProcessIndex, lastVisibleProcessIndex)
	pickDownArrow := len(procs) > 0 && (u.pickedLine == nil || *u.pickedLine < maxPickableIndex)

	u.renderHeaderHints(x0+2+len(byProcess)+3, y0, x1-2, pickDownArrow, pickUpArrow)

	renderLegend(u.screen, u.theme, y1, x1)
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
