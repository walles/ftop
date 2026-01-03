package ptop

import (
	"fmt"
	"slices"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/sysload"
	"github.com/walles/ptop/internal/ui"
)

func renderMemoryUsage(screen twin.Screen) {
	sysload, err := sysload.GetSysload()
	if err != nil {
		// FIXME: Handle this better. What would the user want here?
		panic(err)
	}

	ramUsePercent := float64(sysload.RamUsedBytes) / float64(sysload.RamTotalBytes) * 100.0
	description := fmt.Sprintf("RAM Use: %.0f%%  [%sB / %sB]",
		ramUsePercent,
		ui.FormatMemory(int64(sysload.RamUsedBytes)),
		ui.FormatMemory(int64(sysload.RamTotalBytes)),
	)

	colorLoadBarMin := twin.NewColorHex(0x000000)    // FIXME: Get this from the theme
	colorLoadBarMaxRAM := twin.NewColorHex(0x2020ff) // FIXME: Get this from the theme
	memoryRamp := ui.NewColorRamp(0.0, 1.0, colorLoadBarMin, colorLoadBarMaxRAM)

	width, _ := screen.Size()
	loadBar := ui.NewLoadBar(2, width-2, memoryRamp)

	runes := []rune(description)
	for column := 2; column < width-2; column++ {
		char := ' '
		if column-2 < len(runes) {
			char = runes[column-2]
		}

		style := twin.StyleDefault
		if !slices.Contains([]rune{' ', '[', '/', ']'}, char) {
			style = style.WithAttr(twin.AttrBold)
		}
		loadBar.SetBgColor(&style, column, ramUsePercent/100.0)

		screen.SetCell(column, 2, twin.StyledRune{Rune: char, Style: style})
	}
}
