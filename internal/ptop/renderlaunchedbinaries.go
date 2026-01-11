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

	x0s := []int{}     // Starting screen X positions for each column
	xAfters := []int{} // Screen X positions after each column

	x0 := 1
	for rowIndex, path := range launchSlices {
		x := x0
		y := y0 + 1 + rowIndex
		if y >= y1 {
			// Reached bottom of our allocated area
			return
		}

		hidingRedundantColumns := true
		for column, node := range path {
			if hidingRedundantColumns {
				if rowIndex > 0 {
					// Check is we're still drawing the same column as the row above
					rowAbove := launchSlices[rowIndex-1]
					if column < len(rowAbove) && node.Command == rowAbove[column].Command {
						// Same command as above, skip drawing this column
						x = xAfters[column]
						continue
					}
				}

				hidingRedundantColumns = false
				x0s = x0s[:column]
				xAfters = xAfters[:column]
			}

			// Invariant: No more columns to hide, draw this!

			if x > x0 {
				// Not the first column, start with an arrow as separator
				x += drawText(screen, x, y, rightBorder, "->", twin.StyleDefault)
			}

			style := twin.StyleDefault
			if node.LaunchCount > 0 {
				style = style.WithAttr(twin.AttrBold)
			}

			x0s = append(x0s, x)
			x += drawText(screen, x, y, rightBorder, node.Command, style)

			if node.LaunchCount > 0 {
				x += drawText(screen, x, y, rightBorder, "("+strconv.Itoa(node.LaunchCount)+")", twin.StyleDefault)
			}
			xAfters = append(xAfters, x)
		}
	}
}
