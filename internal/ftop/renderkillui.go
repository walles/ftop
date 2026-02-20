package ftop

import (
	"fmt"

	"github.com/walles/moor/v2/twin"
)

func (u *Ui) renderKillUi(width int, height int) {
	w, h := u.screen.Size()
	xCenter := w / 2
	yCenter := h / 2
	x0 := xCenter - width/2
	x1 := xCenter + width/2
	y0 := yCenter - height/2
	y1 := yCenter + height/2

	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			u.screen.SetCell(x, y, twin.StyledRune{Rune: ' '})
		}
	}

	killHandler, ok := u.eventHandler.(*eventHandlerKill)
	if !ok {
		panic(fmt.Sprintf("Not a kill handler: %+v", u.eventHandler))
	}

	// "Press k to kill launchd(1)."
	x := x0 + 1
	x += drawText(u.screen, x, y0+1, x1, "Press ", u.theme.PromptActive())
	x += drawText(u.screen, x, y0+1, x1, "k", u.theme.PromptKey())
	x += drawText(u.screen, x, y0+1, x1, " to kill ", u.theme.PromptActive())
	x += drawText(u.screen, x, y0+1, x1,
		killHandler.process.String(),
		twin.StyleDefault.WithForeground(u.theme.HighlightedForeground()),
	)
	u.screen.SetCell(x, y0+1, twin.StyledRune{
		Rune:  '.',
		Style: u.theme.PromptActive(),
	})

	renderFrame(u.screen, u.theme, x0, y0, x1, y1, "Kill process")
}
