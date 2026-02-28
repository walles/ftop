package ftop

import (
	"github.com/walles/ftop/internal/ui"
	"github.com/walles/moor/v2/twin"
)

func (u *Ui) renderProcessInfoPane(y0, y1 int) {
	if u.pickedProcess == nil {
		panic("no process picked")
	}

	width, _ := u.screen.Size()
	x1 := width - 1

	defer renderFrame(u.screen, u.theme, 0, y0, x1, y1, "Process Info")

	// Render the process hierarchy

	bottomUpNames := make([]string, 0)
	for p := u.pickedProcess; p != nil; p = p.Parent() {
		bottomUpNames = append(bottomUpNames, p.Command)
	}

	ramp := ui.NewColorRamp(0, float64(len(bottomUpNames))-1, u.theme.Foreground(), u.theme.FadedForeground())

	hierarchy := make([]twin.StyledRune, 0)
	for i := len(bottomUpNames) - 1; i >= 0; i-- {
		if len(hierarchy) > 0 {
			hierarchy = append(hierarchy, twin.StyledRune{Rune: '⎯', Style: twin.StyleDefault})
			hierarchy = append(hierarchy, twin.StyledRune{Rune: '⎯', Style: twin.StyleDefault})
		}

		name := bottomUpNames[i]
		color := ramp.AtInt(i)
		style := twin.StyleDefault.WithForeground(color)
		for _, r := range name {
			hierarchy = append(hierarchy, twin.StyledRune{Rune: r, Style: style})
		}
	}

	// FIXME: If the hierarchy string is too long, take out a part in the middle
	// and replace it with "…"

	y := y0 + 1
	for i, styledRune := range hierarchy {
		x := i + 1
		u.screen.SetCell(x, y, styledRune)
	}
}
