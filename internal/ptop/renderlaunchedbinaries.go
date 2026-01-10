package ptop

import (
	"strconv"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/processes"
)

func renderLaunchedCommands(screen twin.Screen, launches *processes.LaunchNode, y0, y1 int) {
	width, _ := screen.Size()
	rightBorder := width - 1
	defer renderFrame(screen, 0, y0, rightBorder, y1, "Launched Commands")

	if launches == nil {
		return
	}

	launchSlices := launches.Flatten()

	x0 := 1
	for rowIndex, path := range launchSlices {
		x := x0
		y := y0 + 1 + rowIndex
		if y >= y1 {
			// Reached bottom of our allocated area
			return
		}

		for _, node := range path {
			if x > x0 {
				x += drawText(screen, x, y, rightBorder, "->", twin.StyleDefault)
			}

			style := twin.StyleDefault
			if node.LaunchCount > 0 {
				style = style.WithAttr(twin.AttrBold)
			}
			x += drawText(screen, x, y, rightBorder, node.Command, style)
			if node.LaunchCount > 0 {
				x += drawText(screen, x, y, rightBorder, "("+strconv.Itoa(node.LaunchCount)+")", twin.StyleDefault)
			}

			if len(node.Children) == 0 {
				break
			}
		}
	}
}
