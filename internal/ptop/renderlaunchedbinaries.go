package ptop

import (
	"strconv"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/processes"
)

func renderLaunchedCommands(screen twin.Screen, launches *processes.LaunchNode, y0, y1 int) {
	width, _ := screen.Size()
	defer renderFrame(screen, 0, y0, width-1, y1, "Launched Commands")

	if launches == nil {
		return
	}

	// Just render the first line for now
	node := launches
	x0 := 1
	x := x0
	y := y0 + 1
	for {
		if x > x0 {
			x += drawText(screen, x, y, width-1, "->", twin.StyleDefault)
		}

		style := twin.StyleDefault
		if node.LaunchCount > 0 {
			style = style.WithAttr(twin.AttrBold)
		}
		x += drawText(screen, x, y, width-1, node.Command, style)
		if node.LaunchCount > 0 {
			x += drawText(screen, x, y, width-1, "("+strconv.Itoa(node.LaunchCount)+")", style)
		}

		if len(node.Children) == 0 {
			break
		}

		// FIXME: This is a tree, how to render all of it rather than just going for the first child?
		node = node.Children[0]
	}

	renderFrame(screen, 0, y0, width-1, y1, "Launched Commands")
}
