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

// Render the given processes to the given screen, ordered by CPU usage.
func RenderByCpu(processes []Process, screen twin.Screen) {
	processes = ByCpuUsage(processes)
	width, height := screen.Size()

	screen.Clear()

	table := [][]string{
		{
			// These first ones are for the per-process table
			"PID", "COMMAND", "USERNAME", "CPU", "CPUTIME", "RAM",
			// These columns are for the per-user table
			"USERNAME", "CPU", "RAM"},
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
			p.RamPercentString(),
		})
	}

	byUser := aggregateByUser(processes)
	slices.SortFunc(byUser, func(i, j userStats) int {
		byCpuTime := cmp.Compare(i.cpuTime, j.cpuTime)
		if byCpuTime != 0 {
			return -byCpuTime
		}

		// Fall back on process count if the CPU times are equal
		return -cmp.Compare(i.processCount, j.processCount)
	})
	for i, u := range byUser {
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

	colorBg := twin.NewColor24Bit(0, 0, 0) // FIXME: Get this fallback from the theme
	if screen.TerminalBackground() != nil {
		colorBg = *screen.TerminalBackground()
	}

	colorTop := twin.NewColorHex(0xffffff) // FIXME: Get this from the theme
	colorBottom := colorTop.Mix(colorBg, 0.66)
	// 1.0 = ignore the header line
	topBottomRamp := ui.NewColorRamp(colorTop, colorBottom, 1.0, float64(len(table)-1))

	maxCpuTime := time.Duration(0)
	for _, p := range processes {
		if p.cpuTime != nil && *p.cpuTime > maxCpuTime {
			maxCpuTime = *p.cpuTime
		}
	}

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
			if char == '│' {
				// Divider, make it less prominent and don't fade it out
				// FIXME: Get the divider color from the theme
				style = style.WithForeground(twin.NewColorHex(0x404080))
			}
			screen.SetCell(x, rowIndex, twin.StyledRune{Rune: char, Style: style})
			x++
		}
	}

	screen.Show()
}

func aggregateByUser(processes []Process) []userStats {
	userMap := make(map[string]userStats)
	for _, p := range processes {
		stats, exists := userMap[p.username]
		if !exists {
			stats = userStats{username: p.username}
		}

		if p.cpuTime != nil {
			stats.cpuTime += *p.cpuTime
		}
		stats.rssKb += p.rssKb

		stats.processCount++

		userMap[p.username] = stats
	}

	var returnMe []userStats
	for _, stats := range userMap {
		returnMe = append(returnMe, stats)
	}

	return returnMe
}
