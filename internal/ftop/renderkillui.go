package ftop

import "github.com/walles/moor/v2/twin"

func (u *Ui) renderKillUi(width int, height int) {
	w, h := u.screen.Size()
	xCenter := w / 2
	yCenter := h / 2
	x0 := xCenter - width/2
	x1 := xCenter + width/2
	y0 := yCenter - height/2
	y1 := yCenter + height/2

	FIXME: Add a question to this dialog. Consider line lengths and wrapping.

	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			u.screen.SetCell(x, y, twin.StyledRune{
				Rune:  ' ',
				Style: twin.StyleDefault.WithBackground(u.theme.Foreground()),
			})
		}
	}

	renderFrame(u.screen, u.theme, x0, y0, x1, y1, "Kill process")
}
