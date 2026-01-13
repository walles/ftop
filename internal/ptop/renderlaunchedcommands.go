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

	// "" is the empty prefix for the root node
	renderLaunchedCommand(screen, "", launches, 1, y0+1, rightBorder-1, y1-1)
}

// Returns the next Y position to write to after rendering this node and its children.
func renderLaunchedCommand(screen twin.Screen, prefix string, node *processes.LaunchNode, x, y, xMax, yMax int) int {
	if y > yMax {
		return y
	}

	// Draw the arrow prefix
	x += drawText(screen, x, y, xMax, prefix, twin.StyleDefault)

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

	if len(node.Children) == 0 {
		// No children, we're done
		return y + 1
	}

	// Draw the children
	const arrowHead = "─"
	singleChild := len(node.Children) == 1
	for childIndex, child := range node.Children {
		isLastChild := childIndex == len(node.Children)-1

		var shaft string
		if singleChild {
			shaft = "─"
		} else {
			if childIndex == 0 {
				// First child
				shaft = "┬"
			} else if isLastChild {
				// Last child
				shaft = "└"
			} else {
				// Middle child
				shaft = "├"
			}
		}
		nextY := renderLaunchedCommand(screen, shaft+arrowHead, child, x, y, xMax, yMax)

		if !isLastChild {
			// Draw any intermediate vertical shafts
			for y = y + 1; y < nextY; y++ {
				drawText(screen, x, y, xMax, "│", twin.StyleDefault)
			}
		}

		y = nextY
	}

	return y
}
