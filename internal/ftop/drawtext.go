package ftop

import "github.com/walles/moor/v2/twin"

// Returns the number of cells covered
//
// Text will be printed starting at (x, y) up to but not including clipXexclusive.
//
// FIXME: Should this move into twin?
func drawText(screen twin.Screen, x0 int, y int, clipXexclusive int, text string, style twin.Style) int {
	cellCount := 0
	for _, r := range text {
		x := x0 + cellCount
		if x >= clipXexclusive {
			break
		}

		cellCount += screen.SetCell(x, y, twin.StyledRune{Rune: r, Style: style})
	}

	return cellCount
}
