package ptop

import "github.com/walles/moor/v2/twin"

func renderIOLoad(screen twin.Screen) {
	column := 2
	for _, char := range "IO Load:      [422KB/s / 2781KB/s] disk0 (this row is fake)" {
		screen.SetCell(column, 3, twin.StyledRune{Rune: char, Style: twin.StyleDefault})
		column++
	}
}
