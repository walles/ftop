package ptop

import (
	"strings"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/io"
	"github.com/walles/ptop/internal/ui"
)

// Renders max current device BPS vs highest measured BPS
func renderIOLoad(ioStats []io.Stat, screen twin.Screen, width int) {
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

	x1 := width - 1

	x := 2
	y := 3
	x += drawText(screen, x, y, x1, "IO Load:      ", twin.StyleDefault.WithAttr(twin.AttrBold))
	x += drawText(screen, x, y, x1, "[", twin.StyleDefault)
	x += drawText(screen, x, y, x1, bpsStringWithTrailingB, twin.StyleDefault.WithAttr(twin.AttrBold))
	x += drawText(screen, x, y, x1, " / ", twin.StyleDefault)
	x += drawText(screen, x, y, x1, watermarkStringWithTrailingB, twin.StyleDefault)
	x += drawText(screen, x, y, x1, "] ", twin.StyleDefault)
	x += drawText(screen, x, y, x1, maxDevice, twin.StyleDefault.WithAttr(twin.AttrBold)) //nolint:ineffassign
}
