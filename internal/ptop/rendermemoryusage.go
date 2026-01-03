package ptop

import "github.com/walles/moor/v2/twin"

func renderMemoryUsage(screen twin.Screen) {
	column := 2
	for _, char := range "RAM Use: 60%  [19GB / 32GB] (this row is fake)" {
		screen.SetCell(column, 2, twin.StyledRune{Rune: char, Style: twin.StyleDefault})
		column++
	}
}
