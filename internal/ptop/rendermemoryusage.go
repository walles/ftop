package ptop

import (
	"fmt"

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

	// FIXME: Add a RAM load bar

	column := 2
	for _, char := range description {
		screen.SetCell(column, 2, twin.StyledRune{Rune: char, Style: twin.StyleDefault})
		column++
	}
}
