package ftop

import (
	"github.com/walles/moor/v2/twin"
)

func (ui *Ui) renderFilterPrompt(active bool, x0 int, y int, x1 int) {
	x := x0

	if active {
		x += ui.screen.SetCell(x, y, twin.NewStyledRune('F', twin.StyleDefault.WithForeground(ui.theme.Foreground()).WithAttr(twin.AttrReverse)))
		x += drawText(ui.screen, x, y, x1, "ilter", twin.StyleDefault.WithForeground(ui.theme.Foreground()).WithAttr(twin.AttrDim).WithAttr(twin.AttrUnderline))
		ui.screen.SetCell(x, y, twin.StyledRune{
			Style: twin.StyleDefault.WithForeground(ui.theme.HighlightedForeground()).WithAttr(twin.AttrBold),
			Rune:  '⏎',
		})
	} else {
		x += ui.screen.SetCell(x, y, twin.StyledRune{
			Style: twin.StyleDefault.WithForeground(ui.theme.HighlightedForeground()),
			Rune:  'F',
		})
		drawText(ui.screen, x, y, x1, "ilter", twin.StyleDefault.WithForeground(ui.theme.Foreground()).WithAttr(twin.AttrDim))
	}
}
