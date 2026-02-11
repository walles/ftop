package ftop

import (
	"github.com/walles/moor/v2/twin"
)

func (ui *Ui) renderFilterPrompt(x0 int, y int, x1 int) {
	x := x0

	_, isEditingFilter := ui.eventHandler.(*eventHandlerFilter)

	if len(ui.filter) == 0 {
		// No filter
		if isEditingFilter {
			// No filter, but in edit mode
			x += ui.screen.SetCell(x, y, twin.NewStyledRune('F', twin.StyleDefault.WithForeground(ui.theme.Foreground()).WithAttr(twin.AttrReverse)))
			x += drawText(ui.screen, x, y, x1, "ilter", twin.StyleDefault.WithForeground(ui.theme.Foreground()).WithAttr(twin.AttrDim).WithAttr(twin.AttrUnderline))
			ui.screen.SetCell(x, y, twin.StyledRune{
				Style: twin.StyleDefault.WithForeground(ui.theme.HighlightedForeground()).WithAttr(twin.AttrBold),
				Rune:  '⏎',
			})
		} else {
			// No filter, not in edit mode
			x += ui.screen.SetCell(x, y, twin.StyledRune{
				Style: twin.StyleDefault.WithForeground(ui.theme.HighlightedForeground()),
				Rune:  'F',
			})
			drawText(ui.screen, x, y, x1, "ilter", twin.StyleDefault.WithForeground(ui.theme.Foreground()).WithAttr(twin.AttrDim))
		}
	} else {
		// Have a filter

		if isEditingFilter {
			// Have a filter, and in edit mode. Underline the string.
			x += drawText(ui.screen, x, y, x1, ui.filter, twin.StyleDefault.WithForeground(ui.theme.Foreground()).WithAttr(twin.AttrUnderline))

			// Cursor
			x += ui.screen.SetCell(x, y, twin.StyledRune{
				Style: twin.StyleDefault.WithForeground(ui.theme.Foreground()).WithAttr(twin.AttrReverse),
				Rune:  ' ',
			})

			ui.screen.SetCell(x, y, twin.StyledRune{
				Style: twin.StyleDefault.WithForeground(ui.theme.HighlightedForeground()).WithAttr(twin.AttrBold),
				Rune:  '⏎',
			})
		} else {
			// Have a filter, not in edit mode
			x += drawText(ui.screen, x, y, x1, ui.filter, twin.StyleDefault.WithForeground(ui.theme.Foreground()))
			ui.screen.SetCell(x, y, twin.StyledRune{
				Style: twin.StyleDefault.WithForeground(ui.theme.HighlightedForeground()).WithAttr(twin.AttrBold),
				Rune:  '⌫',
			})
		}
	}
}
