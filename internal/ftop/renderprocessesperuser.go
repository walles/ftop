package ftop

import (
	"fmt"

	"github.com/walles/ftop/internal/themes"
	"github.com/walles/ftop/internal/ui"
	"github.com/walles/ftop/internal/util"
	"github.com/walles/moor/v2/twin"
)

func renderPerUser(screen twin.Screen, theme themes.Theme, x0, y0, x1, y1 int, table [][]string, widths []int, users []userStats) {
	widths = widths[6:] // Skip the per-process columns

	// Formats are "%5.5s" or "%-5.5s", where "5.5" means "pad and truncate to
	// 5", and the "-" means left-align.
	formatString := fmt.Sprintf("%%-%d.%ds %%%d.%ds %%%d.%ds",
		widths[0], widths[0],
		widths[1], widths[1],
		widths[2], widths[2],
	)

	memoryRamp := ui.NewColorRamp(0.0, 1.0, theme.LoadBarMin(), theme.LoadBarMaxRam())
	cpuRamp := ui.NewColorRamp(0.0, 1.0, theme.LoadBarMin(), theme.LoadBarMaxCpu())

	// +1 = ignore top border
	topBottomRamp := ui.NewColorRamp(float64(y0+1), float64(y1-1), theme.Foreground(), theme.FadedForeground())

	usernameColumn0 := x0 + 1                          // Screen column
	usernameColumnN := usernameColumn0 + widths[0] - 1 // Screen column
	currentUsername := util.GetCurrentUsername()

	// If y0 = 0 and y1 = 1, then there would be 0 content rows between the
	// borders
	rowsWithoutBorders := y1 - y0 - 1

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

	// Pretend the load bar starts at x0, even though it really starts at x0+1.
	// See the NewOverlappingLoadBars call in renderprocesses.go for details.
	cpuAndMemBar := ui.NewOverlappingLoadBars(x0, x1-1, cpuRamp, memoryRamp)

	//
	// Render table contents
	//

	for rowIndex, row := range table {
		if rowIndex >= rowsWithoutBorders {
			// No more room
			break
		}

		row = row[6:] // Skip the per-process columns
		line := fmt.Sprintf(formatString,
			row[0], row[1], row[2],
		)

		y := y0 + 1 + rowIndex // screen row

		rowStyle := twin.StyleDefault.WithForeground(topBottomRamp.AtInt(y))

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

			if rowIndex < len(users) {
				user := users[rowIndex]
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

	renderFrame(screen, theme, x0, y0, x1, y1, "By User")
}
