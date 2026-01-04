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
		comparison := cmp.Compare(s1.BytesPerSecond, s2.BytesPerSecond)
		if comparison != 0 {
			return -comparison
		}

		// For great stability at the bottom of the list
		return cmp.Compare(s1.DeviceName, s2.DeviceName)
	})

	for i, stat := range ioStats {
		y := topRow + 2 + i

		bottomContentRow := bottomRow - 1
		if y > bottomContentRow {
			break
		}

		bpsStringWithTrailingB := strings.TrimSuffix(ui.FormatMemory(int64(stat.BytesPerSecond)), "B") + "B/s"

		drawText(screen, 2, y, fmt.Sprintf("%-7s %7s", stat.DeviceName, bpsStringWithTrailingB), twin.StyleDefault)
	}

	renderFrame(screen, topRow, 0, bottomRow, width-1, "IO Stats")
}
