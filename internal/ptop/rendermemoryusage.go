package ptop

import (
	"fmt"
	"slices"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/sysload"
	"github.com/walles/ptop/internal/themes"
	"github.com/walles/ptop/internal/ui"
)

func renderMemoryUsage(screen twin.Screen, theme themes.Theme, width int) {
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

	memoryRamp := ui.NewColorRamp(0.0, 1.0, theme.LoadBarMin(), theme.LoadBarMaxRam())

	runes := []rune(description)
	for column := 2; column < width-2; column++ {
		char := ' '
		if column-2 < len(runes) {
			char = runes[column-2]
		}

		style := twin.StyleDefault.WithForeground(theme.Foreground())
		if !slices.Contains([]rune{' ', '[', '/', ']'}, char) {
			style = style.WithAttr(twin.AttrBold)
		}

		screen.SetCell(column, 2, twin.StyledRune{Rune: char, Style: style})
	}

	loadBar := ui.NewLoadBar(2, width-2, memoryRamp)
	for column := 2; column < width-2; column++ {
		loadBar.SetCellBackground(screen, column, 2, ramUsePercent/100.0)
	}
}
