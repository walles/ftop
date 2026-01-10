package ptop

import "github.com/walles/moor/v2/twin"

func renderLaunchedCommands(screen twin.Screen, y0 int, y1 int) {
	width, _ := screen.Size()

	renderFrame(screen, 0, y0, width-1, y1, "Launched Commands")
}
