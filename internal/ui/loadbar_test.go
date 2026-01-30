package ui

import (
	"testing"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/moor/v2/twin"
)

func TestLoadbar(t *testing.T) {
	screen := twin.NewFakeScreen(6, 1)
	color0 := twin.NewColorHex(0x000000)
	color1 := twin.NewColorHex(0xffffff)
	ramp := NewColorRamp(0.0, 1.0, color0, color1)
	loadBar := NewLoadBar(1, 4, ramp)

	unset := twin.NewColorHex(0xff0000)

	y := 0
	for x := range 6 {
		screen.SetCell(x, y, twin.StyledRune{Style: twin.StyleDefault.WithBackground(unset)})
		loadBar.SetCellBackground(screen, x, y, 0.5)
	}

	// At load fraction 0.5, cells 1 and 2 should be colored
	expectedColors := []twin.Color{
		unset,
		color0,
		color1,
		unset,
		unset,
		unset,
	}

	actualColors := make([]twin.Color, 6)
	for x := range 6 {
		actualColors[x] = screen.GetCell(x, y).Style.Background()
	}

	assert.SlicesEqual(t, actualColors, expectedColors)
}
