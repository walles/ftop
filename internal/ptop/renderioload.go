package ptop

import (
	"fmt"
	"strings"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/io"
	"github.com/walles/ptop/internal/ui"
)

func renderIOLoad(ioStats []io.Stat, screen twin.Screen) {
	maxBytesPerSecond := 0.0
	maxHighWatermark := 0.0
	maxDevice := "N/A"

	for _, stat := range ioStats {
		if stat.BytesPerSecond > maxBytesPerSecond {
			maxBytesPerSecond = stat.BytesPerSecond
			maxDevice = stat.DeviceName
		}
		if stat.HighWatermark > maxHighWatermark {
			maxHighWatermark = stat.HighWatermark
		}
	}

	bpsStringWithTrailingB := strings.TrimSuffix(ui.FormatMemory(int64(maxBytesPerSecond)), "B") + "B/s"
	watermarkStringWithTrailingB := strings.TrimSuffix(ui.FormatMemory(int64(maxHighWatermark)), "B") + "B/s"

	// FIXME: Style the output

	description := fmt.Sprintf("IO Load:      [%s / %s] %s",
		bpsStringWithTrailingB,
		watermarkStringWithTrailingB,
		maxDevice,
	)

	column := 2
	for _, char := range description {
		screen.SetCell(column, 3, twin.StyledRune{Rune: char, Style: twin.StyleDefault})
		column++
	}
}
