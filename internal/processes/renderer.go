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

type sectionType int

const (
	sectionTypeCpu sectionType = iota
	sectionTypeMemory
)

func Render(processes []Process, screen twin.Screen) {
	width, height := screen.Size()

	// Decide on section heights
	heightWithoutPadding := height - 4 // 4 = top and bottom frame around each section
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
	allInOneTable := append(cpuTable, memTable...)
	widths := ui.ColumnWidths(allInOneTable, width-4) // 4 = left and right frame around each section

	// If the CPU section is 0 high:
	// 0: ---- start-of-CPU-section divider ---
	// 1: ---- end-of-CPU-section divider ---
	// 2: ---- start-of-Memory-section divider ----
	// 3: Memory section starts here
	//
	// So memory section always starts at cpuHeight + 3
	memSectionStart := len(cpuTable) + 3

	perProcessTableWidth := widths[0] + 1 + widths[1] + 1 + widths[2] + 1 + widths[3] + 1 + widths[4] + 1 + widths[5]
	rightPerProcessBorderColumn := perProcessTableWidth + 1
	leftPerUserBorderColumn := rightPerProcessBorderColumn + 1
	rightPerUserBorderColumn := leftPerUserBorderColumn + widths[6] + 1 + widths[7] + 1 + widths[8] + 1

	// Render!
	screen.Clear()
	renderSection(sectionTypeCpu, cpuTable, widths, processesByCpu, usersByCpu, screen, 1, 1)
	renderSection(sectionTypeMemory, memTable, widths, processesByMem, usersByMem, screen, memSectionStart, 1)

	bottomPerCpuBorderRow := cpuHeight + 2
	bottomPerMemBorderRow := memSectionStart + memHeight + 1
	renderFrame(screen, 0, 0, bottomPerCpuBorderRow, rightPerProcessBorderColumn, "CPU usage by process")
	renderFrame(screen, 0, leftPerUserBorderColumn, bottomPerCpuBorderRow, rightPerUserBorderColumn, "CPU usage by user")
	renderFrame(screen, memSectionStart-1, 0, bottomPerMemBorderRow, rightPerProcessBorderColumn, "Memory usage by process")
	renderFrame(screen, memSectionStart-1, leftPerUserBorderColumn, bottomPerMemBorderRow, rightPerUserBorderColumn, "Memory usage by user")

	screen.Show()
}

func renderFrame(screen twin.Screen, topRow int, leftColumn int, bottomRow int, rightColumn int, title string) {
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
	titleRunes := []rune(title)
	for i, r := range titleRunes {
		if leftColumn+2+i < rightColumn {
			screen.SetCell(leftColumn+2+i, topRow, twin.StyledRune{Rune: r, Style: dividerStyle})
		}
	}
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

func renderSection(sectionType sectionType, table [][]string, widths []int, processes []Process, users []userStats, screen twin.Screen, firstScreenRow int, firstScreenColumn int) {
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

	var maxPerProcess float64
	for _, p := range processes {
		value := getProcessValue(p, sectionType)
		if value > maxPerProcess {
			maxPerProcess = value
		}
	}

	var maxPerUser float64
	for _, u := range users {
		value := getUserValue(u, sectionType)
		if value > maxPerUser {
			maxPerUser = value
		}
	}

	// Indices with the divider is at 5
	// 0123 │ 67
	//
	// So the width of the per-process table in this case is 4
	perProcessTableWidth := widths[0] + 1 + widths[1] + 1 + widths[2] + 1 + widths[3] + 1 + widths[4] + 1 + widths[5]

	perUserTableWidth := widths[6] + 1 + widths[7] + 1 + widths[8]
	perUserTableStart := widths[0] + 1 + widths[1] + 1 + widths[2] + 1 + widths[3] + 1 + widths[4] + 1 + widths[5] + 2 // +2 for the "||" divider

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
			if x < perProcessTableWidth && rowIndex > 0 && maxPerProcess > 0 {
				// On the left side, add per-process load bars
				process := processes[rowIndex-1]
				loadFraction := getProcessValue(process, sectionType) / maxPerProcess
				setLoadBarStyle(&style, x, loadFraction, perProcessTableWidth, colorBg, loadBarRamp)
			} else if x >= perUserTableStart && rowIndex > 0 && rowIndex <= len(users) && maxPerUser > 0 {
				// On the right side, add per-user load bars
				user := users[rowIndex-1]
				loadFraction := getUserValue(user, sectionType) / maxPerUser
				setLoadBarStyle(&style, x-perUserTableStart, loadFraction, perUserTableWidth, colorBg, loadBarRamp)
			}
			screen.SetCell(firstScreenColumn+x, firstScreenRow+rowIndex, twin.StyledRune{Rune: char, Style: style})
			x++
		}
	}
}

func setLoadBarStyle(style *twin.Style, x int, loadFraction float64, fullWidth int, colorBg twin.Color, loadBarRamp ui.ColorRamp) {
	xf := float64(x)
	loadBarFraction := xf / float64(fullWidth)
	if loadBarFraction < 0.0 || loadBarFraction > 1.0 {
		// Out of bounds, never mind
		return
	}

	loadBarWidth := float64(fullWidth) * loadFraction
	if xf >= loadBarWidth {
		// No load bar here
		return
	}

	remaining := loadBarWidth - xf
	if remaining > 1.0 {
		*style = style.WithBackground(loadBarRamp.AtValue(loadBarFraction))
	} else {
		// Anti-aliasing for the load bar's right edge
		colorLoadBar := loadBarRamp.AtValue(loadBarFraction)
		*style = style.WithBackground(colorBg.Mix(colorLoadBar, remaining))
	}
}

func getProcessValue(p Process, sectionType sectionType) float64 {
	if sectionType == sectionTypeMemory {
		return float64(p.rssKb)
	}

	// Section type CPU
	if p.cpuTime != nil {
		return p.cpuTime.Seconds()
	}

	// No answer, treat as zero
	return 0.0
}

func getUserValue(u userStats, sectionType sectionType) float64 {
	if sectionType == sectionTypeMemory {
		return float64(u.rssKb)
	}

	// Section type CPU
	return u.cpuTime.Seconds()
}
