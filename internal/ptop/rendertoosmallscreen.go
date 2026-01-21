package ptop

import (
	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/themes"
	"github.com/walles/ptop/internal/ui"
)

func renderTooSmallScreen(screen twin.Screen, theme themes.Theme) {
	screen.Clear()

	width, height := screen.Size()
	message := "Make terminal larger"
	if width >= minWidth {
		// Width is fine
		message = "Make terminal taller"
	}
	if height >= minHeight {
		// Height is fine
		message = "Make terminal wider"
	}
	x := (width - len(message)) / 2

	topBottomTextRamp := ui.NewColorRamp(0, float64(height-1), theme.Background(), theme.Foreground(), theme.Background())

	for y := range height {
		color := topBottomTextRamp.AtInt(y)
		drawText(screen, x, y, width, message, twin.StyleDefault.WithForeground(color))
	}

	red := twin.NewColorHex(0xff0000)

	if width < minWidth {
		// Red left and right borders
		topBottomRamp := ui.NewColorRamp(0, float64(height-1), theme.Background(), red, theme.Background())

		for y := range height {
			color := topBottomRamp.AtInt(y)

			screen.SetCell(0, y, twin.StyledRune{
				Style: twin.StyleDefault.WithForeground(color),
				Rune:  '<',
			})
			screen.SetCell(width-1, y, twin.StyledRune{
				Style: twin.StyleDefault.WithForeground(color),
				Rune:  '>',
			})
		}
	}

	if height < minHeight {
		// Red top and bottom borders
		topBottomRamp := ui.NewColorRamp(0, float64(width-1), theme.Background(), red, theme.Background())

		for x := range width {
			color := topBottomRamp.AtInt(x)

			screen.SetCell(x, 0, twin.StyledRune{
				Style: twin.StyleDefault.WithForeground(color),
				Rune:  '^',
			})
			screen.SetCell(x, height-1, twin.StyledRune{
				Style: twin.StyleDefault.WithForeground(color),
				Rune:  'v',
			})
		}
	}

	screen.Show()
}
