package ptop

import "github.com/walles/moor/v2/twin"

func renderSysload(screen twin.Screen) {

	column := 2
	for _, char := range "Sysload: 1.4  [8 cores | 16 virtual]  [15m history: ⢸⣿⣿⣿⣿⣿⣿⣷] (this row is fake)" {
		screen.SetCell(column, 1, twin.StyledRune{Rune: char, Style: twin.StyleDefault})
		column++
	}
}
