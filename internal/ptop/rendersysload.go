package ptop

import (
	"fmt"
	"math"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/sysload"
	"github.com/walles/ptop/internal/ui"
)

func renderSysload(screen twin.Screen) {
	sysload, err := sysload.GetSysload()
	if err != nil {
		// FIXME: Handle this better. What would the user want here?
		panic(err)
	}

	// FIXME: Style the string nicely

	description := fmt.Sprintf("Sysload: %.1f  [%d cores | %d virtual]  [15m history: %s]",
		sysload.LoadAverage1M,
		sysload.CpuCoresPhysical,
		sysload.CpuCoresLogical,
		averagesToGraphString(sysload.LoadAverage1M, sysload.LoadAverage5M, sysload.LoadAverage15M),
	)

	width, _ := screen.Size()

	runes := []rune(description)
	for column := 2; column < width-2; column++ {
		char := ' '
		if column-2 < len(runes) {
			char = runes[column-2]
		}

		screen.SetCell(column, 1, twin.StyledRune{Rune: char, Style: twin.StyleDefault})
	}

	colorLoadBarMin := twin.NewColorHex(0x000000)    // FIXME: Get this from the theme
	colorLoadBarMaxCPU := twin.NewColorHex(0x801020) // FIXME: Get this from the theme
	cpuRamp := ui.NewColorRamp(0.0, 1.0, colorLoadBarMin, colorLoadBarMaxCPU)
	loadBar := ui.NewLoadBar(2, width-2, cpuRamp)

	for column := 2; column < width-2; column++ {
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
