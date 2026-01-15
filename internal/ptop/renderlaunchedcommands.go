package ptop

import (
	"strconv"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/processes"
	"github.com/walles/ptop/internal/themes"
	"github.com/walles/ptop/internal/ui"
)

func renderLaunchedCommands(screen twin.Screen, theme themes.Theme, launches *processes.LaunchNode, y0, y1 int) {
	width, _ := screen.Size()
	rightBorder := width - 1
	defer renderFrame(screen, theme, 0, y0, rightBorder, y1, "Launched Commands")

	if launches == nil {
		return
	}

	firstLaunchLine := y0 + 1 // Screen row number
	lastLaunchLine := y1 - 1  // Screen row number
	topBottomRamp := ui.NewColorRamp(float64(firstLaunchLine), float64(lastLaunchLine), theme.Foreground(), theme.FadedForeground())

	// "" is the empty prefix for the root node
	renderLaunchedCommand(screen, "", launches, 1, y0+1, rightBorder-1, y1-1, topBottomRamp)
}

// Returns the next Y position to write to after rendering this node and its children.
func renderLaunchedCommand(screen twin.Screen, prefix string, node *processes.LaunchNode, x, y, xMax, yMax int, topBottomRamp ui.ColorRamp) int {
	if y > yMax {
		return y
	}

	style := twin.StyleDefault.WithForeground(topBottomRamp.AtInt(y))

	// Draw the arrow prefix
	x += drawText(screen, x, y, xMax, prefix, style)

	// Render the command name
	textStyle := style
	if node.LaunchCount > 0 {
		textStyle = textStyle.WithAttr(twin.AttrBold)
	}
	x += drawText(screen, x, y, xMax, node.Command, textStyle)

	if node.LaunchCount > 0 {
		// Render the launch count
		launchCountText := "(" + strconv.Itoa(node.LaunchCount) + ")"
		x += drawText(screen, x, y, xMax, launchCountText, style)
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
		nextY := renderLaunchedCommand(screen, shaft+arrowHead, child, x, y, xMax, yMax, topBottomRamp)

		if !isLastChild {
			// Draw any intermediate vertical shafts
			for y = y + 1; y < nextY; y++ {
				style := twin.StyleDefault.WithForeground(topBottomRamp.AtInt(y))
				drawText(screen, x, y, xMax, "│", style)
			}
		}

		y = nextY
	}

	return y
}
