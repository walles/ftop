package ftop

import (
	"fmt"

	"github.com/walles/moor/v2/twin"
)

func (u *Ui) renderKillUi(nextToScreenRow int) {
	w, h := u.screen.Size()

	// Dialog dimensions
	width := w - 6
	height := 5 // 3 content lines + 2 border lines

	// Horizontal positioning (centered)
	x0 := 3
	x1 := x0 + width - 1

	// Vertical positioning (above or below nextToScreenRow with 1 row gap)
	// Decide based on where there's more room
	roomAbove := nextToScreenRow - 1     // Rows available above (with 1 gap)
	roomBelow := h - nextToScreenRow - 2 // Rows available below (with 1 gap)

	var y0, y1 int
	if roomAbove >= roomBelow {
		// Place above
		y1 = nextToScreenRow - 2 // -2 for the gap
		y0 = y1 - (height - 1)
	} else {
		// Place below
		y0 = nextToScreenRow + 2 // +2 for the gap
		y1 = y0 + (height - 1)
	}

	// Clear the frame
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			u.screen.SetCell(x, y, twin.StyledRune{Rune: ' '})
		}
	}

	killer, ok := u.eventHandler.(*eventHandlerKill)
	if !ok {
		panic(fmt.Sprintf("Not a kill handler: %+v", u.eventHandler))
	}

	excuse := killer.getExcuse()
	if excuse == "" {
		// Kill not attempted yet, tell user we are awaiting confirmation

		// "Press k to kill launchd(1)."
		x := x0 + 1
		x += drawText(u.screen, x, y0+1, x1, "Press ", u.theme.PromptActive())
		x += drawText(u.screen, x, y0+1, x1, "k", u.theme.PromptKey())
		x += drawText(u.screen, x, y0+1, x1, " to kill ", u.theme.PromptActive())
		x += drawText(u.screen, x, y0+1, x1,
			killer.process.String(),
			twin.StyleDefault.WithForeground(u.theme.HighlightedForeground()),
		)
		u.screen.SetCell(x, y0+1, twin.StyledRune{
			Rune:  '.',
			Style: u.theme.PromptActive(),
		})
	} else {
		// We have some excuse, tell the user the kill failed

		// "Failed to kill launchd(1): permission denied."
		// ""
		// "Press any key to continue."
		x := x0 + 1
		y := y0 + 1
		x += drawText(u.screen, x, y, x1, "Failed to kill "+killer.process.String()+": ", twin.StyleDefault)
		drawText(u.screen, x, y, x1,
			excuse,
			twin.StyleDefault.WithForeground(u.theme.HighlightedForeground()),
		)

		x = x0 + 1
		y += 2
		x += drawText(u.screen, x, y, x1, "Press ", u.theme.PromptActive())
		x += drawText(u.screen, x, y, x1, "any key", u.theme.PromptKey())
		drawText(u.screen, x, y, x1, " to continue.", u.theme.PromptActive())
	}

	renderFrame(u.screen, u.theme, x0, y0, x1, y1, "Kill process")
}
