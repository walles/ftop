package ptop

import (
	"fmt"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/sysload"
)

func renderSysload(screen twin.Screen) {
	sysload, err := sysload.GetSysload()
	if err != nil {
		// FIXME: Handle this better. What would the user want here?
		panic(err)
	}

	// FIXME: Fill in sysload values as well

	description := fmt.Sprintf("Sysload: 1.4  [%d cores | %d virtual]  [15m history: ⢸⣿⣿⣿⣿⣿⣿⣷] (this row is mostly fake)",
		sysload.CpuCoresPhysical,
		sysload.CpuCoresLogical,
	)

	width, _ := screen.Size()

	runes := []rune(description)
	for column := 2; column < width-2; column++ {
		char := ' '
		if column-2 < len(runes) {
			char = runes[column-2]
		}

		screen.SetCell(column, 1, twin.StyledRune{Rune: char, Style: twin.StyleDefault})
	}
}
