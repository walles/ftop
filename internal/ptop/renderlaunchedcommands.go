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

	renderLaunchedCommand(screen, launches, 1, y0+1, rightBorder-1, y1-1)
}

// Returns the next Y position to write to after rendering this node and its children.
func renderLaunchedCommand(screen twin.Screen, node *processes.LaunchNode, x, y, xMax, yMax int) int {
	if y > yMax {
		return y
	}

	const x0 = 1 // Leftmost position for text, just insie the border

	if x > x0 {
		// This is not the root node, draw the arrow prefix

		// FIXME: Use the right character based on position in tree
		shaft := "─"

		head := "▶"
		x += drawText(screen, x, y, xMax, shaft+head, twin.StyleDefault)
	}

	// Render the command name
	textStyle := twin.StyleDefault
	if node.LaunchCount > 0 {
		textStyle = textStyle.WithAttr(twin.AttrBold)
	}
	x += drawText(screen, x, y, xMax, node.Command, textStyle)

	if node.LaunchCount > 0 {
		// Render the launch count
		launchCountText := "(" + strconv.Itoa(node.LaunchCount) + ")"
		x += drawText(screen, x, y, xMax, launchCountText, twin.StyleDefault)
	}

	// Draw the children
	for childIndex, child := range node.Children {
		y = renderLaunchedCommand(screen, child, x, y+childIndex, xMax, yMax)
	}

	return y
}
