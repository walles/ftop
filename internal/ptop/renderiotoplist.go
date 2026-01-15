package ptop

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/io"
	"github.com/walles/ptop/internal/themes"
	"github.com/walles/ptop/internal/ui"
)

func renderIoTopList(screen twin.Screen, theme themes.Theme, ioStats []io.Stat, x0, y0, x1, y1 int) {
	slices.SortFunc(ioStats, func(s1, s2 io.Stat) int {
		comparison := cmp.Compare(s1.HighWatermark, s2.HighWatermark)
		if comparison != 0 {
			return -comparison
		}

		// For great stability at the bottom of the list
		return cmp.Compare(s1.DeviceName, s2.DeviceName)
	})

	firstIoLine := y0 + 1 // Screen row number
	lastIoLine := y1 - 1  // Screen row number
	topBottomRamp := ui.NewColorRamp(float64(firstIoLine), float64(lastIoLine), theme.Foreground(), theme.FadedForeground())

	for i, stat := range ioStats {
		y := firstIoLine + i

		bottomContentRow := lastIoLine
		if y > bottomContentRow {
			break
		}

		bpsStringWithTrailingB := strings.TrimSuffix(ui.FormatMemory(int64(stat.BytesPerSecond)), "B") + "B/s"

		paddedDeviceName := fmt.Sprintf("%-7s ", stat.DeviceName)
		x := x0 + 1
		x += drawText(
			screen,
			x,
			y,
			x1,
			paddedDeviceName,
			twin.StyleDefault.WithForeground(topBottomRamp.AtInt(y)).WithAttr(twin.AttrBold))
		x += drawText( // nolint:ineffassign
			screen,
			x,
			y,
			x1,
			fmt.Sprintf("%7s", bpsStringWithTrailingB),
			twin.StyleDefault.WithForeground(topBottomRamp.AtInt(y)),
		)
	}

	maxPeak := 0.0
	for _, stat := range ioStats {
		if stat.HighWatermark > maxPeak {
			maxPeak = stat.HighWatermark
		}
	}

	if maxPeak != 0.0 {
		// Draw the load bars
		ioRamp := ui.NewColorRamp(0.0, 1.0, theme.LoadBarMin(), theme.LoadBarMaxIO())

		for i, stat := range ioStats {
			y := firstIoLine + i

			if y > lastIoLine {
				break
			}

			loadBar := ui.NewLoadBar(x0+1, x1-1, ioRamp)
			loadBar.SetWatermark(stat.HighWatermark / maxPeak)
			for x := x0 + 1; x < x1; x++ {
				loadBar.SetCellBackground(screen, x, y, stat.BytesPerSecond/maxPeak)
			}
		}
	}

	renderFrame(screen, theme, x0, y0, x1, y1, "IO")
}
