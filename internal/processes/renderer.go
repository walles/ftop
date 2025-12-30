package processes

import (
	"fmt"

	"github.com/walles/moor/v2/twin"
)

// Render the given processes to the given screen
func Render(processes []*Process, screen twin.Screen) {
	_, height := screen.Size()

	screen.Clear()
	for i, p := range processes {
		if i >= height {
			break
		}

		line := fmt.Sprintf("%-5d %-28s %-14s %-3s %-3s", p.pid, p.command, p.username, p.CpuPercentString(), p.CpuTimeString())

		for x, char := range line {
			screen.SetCell(x, i, twin.StyledRune{Rune: char})
		}
	}
	screen.Show()
}
