package ptop

import "github.com/walles/moor/v2/twin"

func renderLaunchedBinaries(screen twin.Screen, topRow int, bottomRow int) {
	width, _ := screen.Size()

	renderFrame(screen, topRow, 0, bottomRow, width-1, "Launched Binaries")
}
