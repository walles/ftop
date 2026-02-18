package ftop

import (
	"github.com/walles/moor/v2/twin"
)

func (ui *Ui) renderHeaderHints(x0 int, y int, x1 int, pickDownArrow bool, pickUpArrow bool) {
	x := x0

	x = ui.renderPickAProcPrompt(x, y, x1, pickDownArrow, pickUpArrow)
	x += 3
	x = ui.renderKillPrompt(x, y, x1)
	x += 3
	ui.renderFilterPrompt(x, y, x1)
}

// Returns the first empty x coordinate after the rendered prompt
func (ui *Ui) renderPickAProcPrompt(x0 int, y int, x1 int, pickDownArrow bool, pickUpArrow bool) int {
	x := x0

	downStyle := ui.theme.PromptKey()
	if !pickDownArrow {
		downStyle = ui.theme.PromptPassive()
	}
	x += ui.screen.SetCell(x, y, twin.StyledRune{
		Style: downStyle,
		Rune:  '↓',
	})

	style := ui.theme.PromptActive()
	if ui.pickedLine == nil {
		style = ui.theme.PromptPassive()
	}
	x += drawText(ui.screen, x, y, x1, "Pick", style)

	upStyle := ui.theme.PromptKey()
	if !pickUpArrow {
		upStyle = ui.theme.PromptPassive()
	}
	x += ui.screen.SetCell(x, y, twin.StyledRune{
		Style: upStyle,
		Rune:  '↑',
	})

	return x
}

func (ui *Ui) renderKillPrompt(x0 int, y int, x1 int) int {
	x := x0

	if ui.pickedLine == nil {
		x += drawText(ui.screen, x0, y, x1, "Kill", ui.theme.PromptPassive())
	} else {
		x += ui.screen.SetCell(x, y, twin.StyledRune{
			Style: ui.theme.PromptKey(),
			Rune:  'K',
		})
		x += drawText(ui.screen, x, y, x1, "ill", ui.theme.PromptActive())
	}

	return x
}

func (ui *Ui) renderFilterPrompt(x0 int, y int, x1 int) {
	x := x0

	_, isEditingFilter := ui.eventHandler.(*eventHandlerFilter)

	if len(ui.filter) == 0 {
		// No filter
		if isEditingFilter {
			// No filter, but in edit mode, use plain foreground color to
			// indicate text entry
			style := twin.StyleDefault.WithForeground(ui.theme.Foreground())
			x += ui.screen.SetCell(x, y, twin.NewStyledRune('F', style.WithAttr(twin.AttrReverse)))
			x += drawText(ui.screen, x, y, x1, "ilter", style.WithAttr(twin.AttrUnderline))
			ui.screen.SetCell(x, y, twin.StyledRune{
				Style: ui.theme.PromptKey(),
				Rune:  '⏎',
			})
		} else {
			// No filter, not in edit mode
			x += ui.screen.SetCell(x, y, twin.StyledRune{
				Style: ui.theme.PromptKey(),
				Rune:  'F',
			})
			drawText(ui.screen, x, y, x1, "ilter", ui.theme.PromptPassive())
		}
	} else {
		// Have a filter

		if isEditingFilter {
			// Have a filter, and in edit mode. Use plain foreground to indicate
			// text entry, and underline the string.
			style := twin.StyleDefault.WithForeground(ui.theme.Foreground())
			x += drawText(ui.screen, x, y, x1, ui.filter, style.WithAttr(twin.AttrUnderline))

			// Cursor
			x += ui.screen.SetCell(x, y, twin.StyledRune{
				Style: style.WithAttr(twin.AttrReverse),
				Rune:  ' ',
			})

			// Fill with underlined characters to match the length of the
			// edit-mode-without-filter text, so the end marker doesn't jump
			// around when the user starts typing.
			lastX := x0 + len("Filter")
			for x < lastX {
				x += ui.screen.SetCell(x, y, twin.StyledRune{
					Style: style.WithAttr(twin.AttrUnderline),
					Rune:  ' ',
				})
			}

			ui.screen.SetCell(x, y, twin.StyledRune{
				Style: ui.theme.PromptKey(),
				Rune:  '⏎',
			})
		} else {
			// Have a filter, not in edit mode. Use plain foreground to indicate
			// edited text.
			style := twin.StyleDefault.WithForeground(ui.theme.Foreground()).WithAttr(twin.AttrUnderline)
			x += drawText(ui.screen, x, y, x1, ui.filter, style)
			ui.screen.SetCell(x, y, twin.StyledRune{
				Style: ui.theme.PromptKey(),
				Rune:  '⌫',
			})
		}
	}
}
