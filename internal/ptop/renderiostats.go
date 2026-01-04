package ptop

import (
	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/io"
)

func renderIoStats(ioStats []io.Stat, screen twin.Screen, topRow int, bottomRow int) {
	width, _ := screen.Size()

	renderFrame(screen, topRow, 0, bottomRow, width-1, "IO Stats")
}
