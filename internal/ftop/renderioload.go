package ftop

import (
	"strings"

	"github.com/walles/ftop/internal/io"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/ftop/internal/ui"
	"github.com/walles/moor/v2/twin"
)

// Renders max current device BPS vs highest measured BPS
func renderIOLoad(screen twin.Screen, theme themes.Theme, ioStats []io.Stat, width int) {
	style := twin.StyleDefault.WithForeground(theme.Foreground())

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
	x += drawText(screen, x, y, x1, "IO Load:      ", style.WithAttr(twin.AttrBold))
	x += drawText(screen, x, y, x1, "[", style)
	x += drawText(screen, x, y, x1, bpsStringWithTrailingB, style.WithAttr(twin.AttrBold))
	x += drawText(screen, x, y, x1, " / ", style)
	x += drawText(screen, x, y, x1, watermarkStringWithTrailingB, style)
	x += drawText(screen, x, y, x1, "] ", style)
	x += drawText(screen, x, y, x1, maxDevice, style.WithAttr(twin.AttrBold)) //nolint:ineffassign
}
