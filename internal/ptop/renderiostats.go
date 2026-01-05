package ptop

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/io"
	"github.com/walles/ptop/internal/ui"
)

func renderIoStats(ioStats []io.Stat, screen twin.Screen, topRow int, bottomRow int) {
	width, _ := screen.Size()

	drawText(screen, 2, topRow+1, "Device  Bytes/s", twin.StyleDefault.WithAttr(twin.AttrBold))

	slices.SortFunc(ioStats, func(s1, s2 io.Stat) int {
		comparison := cmp.Compare(s1.HighWatermark, s2.HighWatermark)
		if comparison != 0 {
			return -comparison
		}

		// For great stability at the bottom of the list
		return cmp.Compare(s1.DeviceName, s2.DeviceName)
	})

	colorBg := twin.NewColor24Bit(0, 0, 0) // FIXME: Get this fallback from the theme
	if screen.TerminalBackground() != nil {
		colorBg = *screen.TerminalBackground()
	}

	colorTop := twin.NewColorHex(0xdddddd) // FIXME: Get this from the theme
	colorBottom := colorTop.Mix(colorBg, 0.33)
	// 1.0 = ignore the header line
	firstIoLine := topRow + 2
	lastIoLine := bottomRow - 1
	topBottomRamp := ui.NewColorRamp(float64(firstIoLine), float64(lastIoLine), colorTop, colorBottom)

	for i, stat := range ioStats {
		y := topRow + 2 + i

		bottomContentRow := bottomRow - 1
		if y > bottomContentRow {
			break
		}

		bpsStringWithTrailingB := strings.TrimSuffix(ui.FormatMemory(int64(stat.BytesPerSecond)), "B") + "B/s"

		drawText(
			screen,
			2,
			y,
			fmt.Sprintf("%-7s %7s", stat.DeviceName, bpsStringWithTrailingB),
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
		colorLoadBarMin := twin.NewColorHex(0x000000)   // FIXME: Get this from the theme
		colorLoadBarMaxIO := twin.NewColorHex(0xd0d020) // FIXME: Get this from the theme
		ioRamp := ui.NewColorRamp(0.0, 1.0, colorLoadBarMin, colorLoadBarMaxIO)

		for i, stat := range ioStats {
			y := topRow + 2 + i

			bottomContentRow := bottomRow - 1
			if y > bottomContentRow {
				break
			}

			loadBar := ui.NewLoadBar(2, width-2, ioRamp)
			loadBar.SetWatermark(stat.HighWatermark / maxPeak)
			for column := 2; column < width-2; column++ {
				loadBar.SetCellBackground(screen, column, y, stat.BytesPerSecond/maxPeak)
			}
		}
	}

	renderFrame(screen, topRow, 0, bottomRow, width-1, "IO")
}
