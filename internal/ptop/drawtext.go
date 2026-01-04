package ptop

import "github.com/walles/moor/v2/twin"

// Returns the number of cells covered
//
// FIXME: Should this move into twin?
func drawText(screen twin.Screen, x int, y int, text string, style twin.Style) int {
	runes := []rune(text)
	cellCount := 0
	for i, r := range runes {
		cellCount += screen.SetCell(x+i, y, twin.StyledRune{Rune: r, Style: style})
	}
	return cellCount
}
