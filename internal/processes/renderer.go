package processes

import (
	"fmt"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/layout"
)

// Render the given processes to the given screen
func Render(processes []*Process, screen twin.Screen) {
	_, height := screen.Size()

	screen.Clear()

	table := [][]string{
		{"PID", "COMMAND", "USERNAME", "CPU", "CPUTIME", "RAM"},
	}
	for i, p := range processes {
		if i >= height {
			break
		}

		table = append(table, []string{
			fmt.Sprintf("%d", p.pid),
			p.command,
			p.username,
			p.CpuPercentString(),
			p.CpuTimeString(),
			p.RamPercentString(),
		})
	}

	widths := layout.ColumnWidths(table)
	formatString := fmt.Sprintf("%%%ds %%-%ds %%-%ds %%%ds %%%ds %%%ds",
		widths[0], widths[1], widths[2], widths[3], widths[4], widths[5],
	)

	for rowIndex, row := range table {
		line := fmt.Sprintf(formatString,
			row[0], row[1], row[2], row[3], row[4], row[5],
		)
		for x, char := range line {
			screen.SetCell(x, rowIndex, twin.StyledRune{Rune: char})
		}
	}

	screen.Show()
}
