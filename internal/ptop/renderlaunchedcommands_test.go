package ptop

import (
	"reflect"
	"strings"
	"testing"

	"github.com/walles/moor/v2/twin"
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

	expected := []string{
		"a┬▶b─▶c─▶d",
		" ├▶e",
		" ├▶f┬▶g",
		" │  └▶h─▶i",
		" └▶j",
	}

	if reflect.DeepEqual(screenRows, expected) {
		// That's what we wanted!
		return
	}

	// Failed, print diagnostics: print each slice element on its own line.
	// First "Expected" with rows below each other, then "Actual" the same way.
	join := func(s []string) string {
		if len(s) == 0 {
			return " (empty)"
		}

		out := ""
		for _, line := range s {
			out += "\n" + line
		}

		return out
	}

	t.Fatalf("\nExpected:%s\n\nActual:%s",
		join(expected), join(screenRows))
}
