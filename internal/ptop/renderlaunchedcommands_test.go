package ptop

import (
	"strings"
	"testing"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/assert"
)

func TestRenderLaunchedCommands(t *testing.T) {
	width, height := 10, 10
	screen := twin.NewFakeScreen(width, height)
	screen.Clear()

	renderLaunchedCommands(screen, nil, 0, height-1)

	screenRows := []string{}
	for y := 1; y < height-1; y++ {
		row := ""
		for x := 1; x < width-1; x++ {
			row += string(screen.GetCell(x, y).Rune)
		}

		row = strings.TrimRight(row, " ")
		if row == "" {
			// No more content lines
			break
		}
		screenRows = append(screenRows, row)
	}

	assert.SlicesEqual(t, screenRows, []string{
		"a┬▶b─▶c─▶d",
		" ├▶e",
		" ├▶f┬▶g",
		" │  └▶h─▶i",
		" └▶j",
	})
}
