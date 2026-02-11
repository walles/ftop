package ftop

import (
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

func renderFilterPrompt(screen twin.Screen, theme themes.Theme, active bool, x0 int, y int, x1 int) {
	x := x0

	if active {
		x += screen.SetCell(x, y, twin.NewStyledRune('F', twin.StyleDefault.WithForeground(theme.Foreground()).WithAttr(twin.AttrReverse)))
		x += drawText(screen, x, y, x1, "ilter", twin.StyleDefault.WithForeground(theme.Foreground()).WithAttr(twin.AttrDim).WithAttr(twin.AttrUnderline))
		screen.SetCell(x, y, twin.StyledRune{
			Style: twin.StyleDefault.WithForeground(theme.HighlightedForeground()).WithAttr(twin.AttrBold),
			Rune:  '⏎',
		})
	} else {
		x += screen.SetCell(x, y, twin.StyledRune{
			Style: twin.StyleDefault.WithForeground(theme.HighlightedForeground()),
			Rune:  'F',
		})
		drawText(screen, x, y, x1, "ilter", twin.StyleDefault.WithForeground(theme.Foreground()).WithAttr(twin.AttrDim))
	}
}
