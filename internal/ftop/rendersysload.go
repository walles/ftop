package ftop

import (
	"fmt"
	"math"

	"github.com/walles/ftop/internal/sysload"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/ftop/internal/ui"
	"github.com/walles/moor/v2/twin"
)

func renderSysload(screen twin.Screen, theme themes.Theme, width int) {
	sysload, err := sysload.GetSysload()
	if err != nil {
		// FIXME: Handle this better. What would the user want here?
		panic(err)
	}

	style := twin.StyleDefault.WithForeground(theme.Foreground())

	x1 := width - 1

	x := 1
	y := 1
	x += drawText(screen, x, y, x1, "Sysload: ", style.WithAttr(twin.AttrBold))

	x += drawText(screen, x, y, x1, fmt.Sprintf("%.1f", sysload.LoadAverage1M), style.WithAttr(twin.AttrBold))

	x += drawText(screen, x, y, x1, "  [", style)
	x += drawText(screen, x, y, x1, fmt.Sprintf("%d cores", sysload.CpuCoresPhysical), style.WithAttr(twin.AttrBold))
	x += drawText(screen, x, y, x1, fmt.Sprintf(" | %d virtual] [15m history: ", sysload.CpuCoresLogical), style)
	brailleStartColumn := x
	averageGraph := averagesToGraphString(sysload.LoadAverage1M, sysload.LoadAverage5M, sysload.LoadAverage15M)
	x += drawText(screen, x, y, x1, averageGraph, style.WithAttr(twin.AttrBold))
	brailleEndColumn := x - 1

	x += drawText(screen, x, y, x1, "]", style) // nolint:ineffassign

	// Text in place, now color the braille graph

	brailleRamp := ui.NewColorRamp(float64(brailleStartColumn), float64(brailleEndColumn),
		theme.FadedForeground(),
		theme.Foreground(),
	)
	for column := brailleStartColumn; column <= brailleEndColumn; column++ {
		cell := screen.GetCell(column, 1)
		style := cell.Style.WithForeground(brailleRamp.AtInt(column)).WithAttr(twin.AttrBold)
		screen.SetCell(column, 1, twin.StyledRune{Rune: cell.Rune, Style: style})
	}

	// Finally, draw the load bar behind our text

	cpuRamp := ui.NewColorRamp(0.0, 1.0, theme.LoadBarMin(), theme.LoadBarMaxCpu())
	loadBar := ui.NewLoadBar(1, width-2, cpuRamp)

	for column := 1; column < width-2; column++ {
		loadBar.SetCellBackground(screen, column, 1, sysload.LoadAverage1M/float64(sysload.CpuCoresLogical))
	}
}

// Take one average and convert it into level 0-3, given a peak value
func averageToLevel(avg float64, peak float64) int {
	level := 3 * avg / peak
	return int(math.Round(level))
}

// Converts three load averages into three levels.
//
// A level is a 0-3 integer value.
//
// This function returns the three leves, plus the peak value the levels are
// based on.
func averagesToLevels(avg1M, avg5M, avg15M float64) (int, int, int, float64) {
	peak := max(avg1M, avg5M, avg15M)
	if peak < 1.0 {
		peak = 1.0
	}

	level1M := averageToLevel(avg1M, peak)
	level5M := averageToLevel(avg5M, peak)
	level15M := averageToLevel(avg15M, peak)

	return level1M, level5M, level15M, peak
}

// Convert three load averages into a unicode string graph.
//
// Each level in the levels array is an integer 0-3. Those levels will be
// represented in the graph by 1-4 dots each.
//
// The returned string will contain two levels per rune.
func averagesToGraphString(level1M, level5M, level15M float64) string {
	l1, l5, l15, _ := averagesToLevels(level1M, level5M, level15M)

	// Collect one padding level to get to an even number of columns (16), 10
	// levels for 15m, 4 levels for 5m, and 1 level for 1m.
	levels := []int{-1}
	for range 10 {
		levels = append(levels, l15)
	}
	for range 4 {
		levels = append(levels, l5)
	}
	levels = append(levels, l1)

	// https://en.wikipedia.org/wiki/Braille_Patterns#Identifying.2C_naming_and_ordering
	leftLevels := []rune{0x00, 0x40, 0x44, 0x46, 0x47}
	rightLevels := []rune{0x00, 0x80, 0xA0, 0xB0, 0xB8}

	graph := ""
	for i := 0; i < len(levels); i += 2 {
		leftLevel := levels[i] + 1
		rightLevel := levels[i+1] + 1

		r := rune(0x2800) + leftLevels[leftLevel] + rightLevels[rightLevel]
		graph += string(r)
	}

	return graph
}
