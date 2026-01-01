package processes

import (
	"cmp"
	"fmt"
	"slices"
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

func Render(processes []Process, screen twin.Screen) {
	width, height := screen.Size()

	// Decide on section heights
	heightWithoutPadding := height - 2 // 2 = divider + footer
	cpuHeight := heightWithoutPadding / 2
	memHeight := heightWithoutPadding - cpuHeight

	// Decide on section contents. "-1" = Leave room for the header row
	processesByCpu := ProcessesByCpuUsage(processes)[:cpuHeight-1]
	usersByCpu := UsersByCpuUsage(processes)[:cpuHeight-1]
	processesByMem := ProcessesByMemoryUsage(processes)[:memHeight-1]
	usersByMem := UsersByMemoryUsage(processes)[:memHeight-1]

	// Adjust heights to what we actually have
	cpuHeight = max(len(processesByCpu), len(usersByCpu))
	memHeight = max(len(processesByMem), len(usersByMem))

	// Figure out column widths
	cpuTable := toTable(processesByCpu, usersByCpu)
	memTable := toTable(processesByMem, usersByMem)
	allInOneTable := append(cpuTable, memTable...) // [1:] to avoid duplicating header row
	widths := ui.ColumnWidths(allInOneTable, width)

	// If the CPU section is 0 high:
	// 0: ----
	// 1: Memory section starts here
	//
	// So memory section always starts at cpuHeight + 1
	memSectionStart := cpuHeight + 1

	// Render!
	screen.Clear()
	renderSection(cpuTable, widths, processesByCpu, usersByCpu, screen, 0)
	// FIXME: Render a divider line
	renderSection(memTable, widths, processesByMem, usersByMem, screen, memSectionStart)
	// FIXME: Render a footer line with instructions
	screen.Show()
}

func toTable(processesByCpu []Process, usersByCpu []userStats) [][]string {
	headerLine := []string{
		// These first ones are for the per-process table
		"PID", "Command", "User name", "CPU", "Time", "RAM",
		// These columns are for the per-user table
		"User name", "CPU", "RAM",
	}

	var table [][]string

	// Header line
	table = append(table, headerLine)

	for i := 0; i < max(len(processesByCpu), len(usersByCpu)); i++ {
		row := make([]string, 0, len(headerLine))

		if i < len(processesByCpu) {
			p := processesByCpu[i]
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

		if i < len(usersByCpu) {
			u := usersByCpu[i]
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

func renderSection(table [][]string, widths []int, processesByCpu []Process, usersByCpu []userStats, screen twin.Screen, firstScreenRow int) {
	// Formats are "%5.5s" or "%-5.5s", where "5.5" means "pad and truncate to
	// 5", and the "-" means left-align.
	formatString := fmt.Sprintf("%%%d.%ds %%-%d.%ds %%-%d.%ds %%%d.%ds %%%d.%ds %%%d.%ds │ %%-%d.%ds %%%d.%ds %%%d.%ds",
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

	dividerColumn := widths[0] + 1 + widths[1] + 1 + widths[2] + 1 + widths[3] + 1 + widths[4] + 1 + widths[5] + 1
	colorDivider := twin.NewColorHex(0x7070a0) // FIXME: Get this from the theme

	colorBg := twin.NewColor24Bit(0, 0, 0) // FIXME: Get this fallback from the theme
	if screen.TerminalBackground() != nil {
		colorBg = *screen.TerminalBackground()
	}

	colorTop := twin.NewColorHex(0xdddddd) // FIXME: Get this from the theme
	colorBottom := colorTop.Mix(colorBg, 0.66)
	// 1.0 = ignore the header line
	topBottomRamp := ui.NewColorRamp(1.0, float64(len(table)-1), colorTop, colorBottom)

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

		x := 0
		for _, char := range line {
			style := rowStyle
			if x == dividerColumn {
				style = style.WithForeground(colorDivider)
			}
			screen.SetCell(x, firstScreenRow+rowIndex, twin.StyledRune{Rune: char, Style: style})
			x++
		}
	}
}

// Render the given processes to the given screen, ordered by CPU usage.
//
// FIXME: Test this when we have fewer processes than the screen is high
// FIXME: Remove this function, it's not used anymore
func RenderByCpu(processes []Process, screen twin.Screen) {
	processes = ProcessesByCpuUsage(processes)
	width, height := screen.Size()

	screen.Clear()

	table := [][]string{
		{
			// These first ones are for the per-process table
			"PID", "Command", "User name", "CPU", "Time", "RAM",
			// These columns are for the per-user table
			"User name", "CPU", "RAM"},
	}

	// Fill in the per-process columns
	for i, p := range processes {
		if i >= height {
			break
		}

		table = append(table, []string{
			fmt.Sprintf("%d", p.pid),
			p.command,
			p.username,
			p.CpuPercentString(),
			p.CpuTimeString(),
			formatMemory(int64(p.rssKb) * 1024),
		})
	}

	// Fill in the per-user columns
	perUser := aggregatePerUser(processes)
	slices.SortFunc(perUser, func(i, j userStats) int {
		byCpuTime := cmp.Compare(i.cpuTime, j.cpuTime)
		if byCpuTime != 0 {
			return -byCpuTime
		}

		// Before we have any CPU times, the count will give a similar ordering
		byProcessCount := cmp.Compare(i.processCount, j.processCount)
		if byProcessCount != 0 {
			return -byProcessCount
		}

		// Sorting by memory usage stabilizes the bottom of the list
		byMemoryUsage := cmp.Compare(i.rssKb, j.rssKb)
		if byMemoryUsage != 0 {
			return -byMemoryUsage
		}

		// Fall back on user names to get a stable result at the very end of the
		// list.
		return cmp.Compare(i.username, j.username)
	})
	for i, u := range perUser {
		tableRowIndex := i + 1 // +1 to account for header row

		// With height 2, the highest allowed tableRowIndex is 1. So if the
		// index is equal to height, we're out of bounds.
		if tableRowIndex >= height {
			// No more room
			break
		}

		if tableRowIndex >= len(table) {
			// Prepend empty per-process data. I don't see how we could ever get
			// here.
			table = append(table, []string{"", "", "", "", "", ""})
		}

		table[tableRowIndex] = append(table[tableRowIndex],
			u.username,
			formatDuration(u.cpuTime),
			formatMemory(1024*int64(u.rssKb)),
		)
	}
	for i := range len(table) {
		if len(table[i]) < 9 {
			// Pad out rows that don't have per-user data
			table[i] = append(table[i], "", "", "")
		}
	}

	// "-10" = the number of between-column-spaces we need, see format string
	// below
	widths := ui.ColumnWidths(table, width-10)

	// Formats are "%5.5s" or "%-5.5s", where "5.5" means "pad and truncate to
	// 5", and the "-" means left-align.
	formatString := fmt.Sprintf("%%%d.%ds %%-%d.%ds %%-%d.%ds %%%d.%ds %%%d.%ds %%%d.%ds │ %%-%d.%ds %%%d.%ds %%%d.%ds",
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

	// FIXME: Why "+1" here? Shouldn't it be "+6" to account for the spaces
	// between the columns?
	dividerColumn := widths[0] + 1 + widths[1] + 1 + widths[2] + 1 + widths[3] + 1 + widths[4] + 1 + widths[5] + 1
	colorDivider := twin.NewColorHex(0x7070a0) // FIXME: Get this from the theme

	colorLoadBarMin := twin.NewColorHex(0x204020) // FIXME: Get this from the theme
	colorLoadBarMid := twin.NewColorHex(0x808020) // FIXME: Get this from the theme
	colorLoadBarMax := twin.NewColorHex(0x801020) // FIXME: Get this from the theme
	loadBarRamp := ui.NewColorRamp(0.0, 1.0, colorLoadBarMin, colorLoadBarMid, colorLoadBarMax)

	colorBg := twin.NewColor24Bit(0, 0, 0) // FIXME: Get this fallback from the theme
	if screen.TerminalBackground() != nil {
		colorBg = *screen.TerminalBackground()
	}

	colorTop := twin.NewColorHex(0xdddddd) // FIXME: Get this from the theme
	colorBottom := colorTop.Mix(colorBg, 0.66)
	// 1.0 = ignore the header line
	topBottomRamp := ui.NewColorRamp(1.0, float64(len(table)-1), colorTop, colorBottom)

	maxPerProcessCpuTime := time.Duration(0)
	for _, p := range processes {
		if p.cpuTime != nil && *p.cpuTime > maxPerProcessCpuTime {
			maxPerProcessCpuTime = *p.cpuTime
		}
	}

	maxPerUserCpuTime := time.Duration(0)
	for _, u := range perUser {
		if u.cpuTime > maxPerUserCpuTime {
			maxPerUserCpuTime = u.cpuTime
		}
	}

	// If there is one user, it will get row index 1, since 0 is the header row.
	// So in this case, len(byUser) is 1, and the max index is 1.
	maxPerUserRowIndex := len(perUser)

	// Indices with the divider is at 5
	// 0123 │ 67
	//
	// So the width of the per-process table in this case is 4
	perProcessTableWidth := dividerColumn - 1

	perUserTableWidth := widths[6] + 1 + widths[7] + 1 + widths[8]
	perUserTableStart := dividerColumn + 2 // +2 to account for the divider and the space after it

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

		x := 0
		for _, char := range line {
			style := rowStyle
			if x == dividerColumn {
				style = style.WithForeground(colorDivider)
			} else if x < perProcessTableWidth && rowIndex > 0 && maxPerProcessCpuTime > 0 {
				// On the left side, add per-process load bars
				loadFraction := processes[rowIndex-1].cpuTime.Seconds() / maxPerProcessCpuTime.Seconds()
				loadBarWidth := float64(perProcessTableWidth) * loadFraction
				xf := float64(x)
				loadBarFraction := xf / float64(perProcessTableWidth)
				if xf < loadBarWidth {
					remaining := loadBarWidth - xf
					if remaining > 1.0 {
						style = style.WithBackground(loadBarRamp.AtValue(loadBarFraction))
					} else {
						// Anti-aliasing for the load bar's right edge
						colorLoadBar := loadBarRamp.AtValue(loadBarFraction)
						style = style.WithBackground(colorBg.Mix(colorLoadBar, remaining))
					}
				}
			} else if x >= perUserTableStart && rowIndex > 0 && rowIndex <= maxPerUserRowIndex && maxPerUserCpuTime > 0 {
				// On the right side, add per-user load bars
				loadFraction := perUser[rowIndex-1].cpuTime.Seconds() / maxPerUserCpuTime.Seconds()
				loadBarWidth := float64(perUserTableWidth) * loadFraction
				xf := float64(x - perUserTableStart) // x coordinate relative to the left edge of the per-user table
				loadBarFraction := xf / float64(perUserTableWidth)
				if xf < loadBarWidth && xf >= 0.0 {
					remaining := loadBarWidth - xf
					if remaining > 1.0 {
						style = style.WithBackground(loadBarRamp.AtValue(loadBarFraction))
					} else {
						// Anti-aliasing for the load bar's right edge
						colorLoadBar := loadBarRamp.AtValue(loadBarFraction)
						style = style.WithBackground(colorBg.Mix(colorLoadBar, remaining))
					}
				}
			}
			screen.SetCell(x, rowIndex, twin.StyledRune{Rune: char, Style: style})
			x++
		}
	}

	screen.Show()
}
