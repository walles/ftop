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

	halfWayY := height / 2
	topToMiddleTextRamp := ui.NewColorRamp(0, float64(halfWayY), theme.Background(), theme.Foreground())
	middleToBottomTextRamp := ui.NewColorRamp(float64(halfWayY+1), float64(height), theme.Foreground(), theme.Background())

	for y := range height {
		var color twin.Color
		if y <= halfWayY {
			color = topToMiddleTextRamp.AtInt(y)
		} else {
			color = middleToBottomTextRamp.AtInt(y)
		}
		drawText(screen, x, y, width, message, twin.StyleDefault.WithForeground(color))
	}

	red := twin.NewColorHex(0xff0000)

	if width < minWidth {
		// Red left and right borders
		topMiddleRamp := ui.NewColorRamp(0, float64(halfWayY), theme.Background(), red)
		middleBottomRamp := ui.NewColorRamp(float64(halfWayY+1), float64(height-1), red, theme.Background())

		for y := range height {
			var color twin.Color
			if y <= halfWayY {
				color = topMiddleRamp.AtInt(y)
			} else {
				color = middleBottomRamp.AtInt(y)
			}
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
		leftMiddleRamp := ui.NewColorRamp(0, float64(width)/2, theme.Background(), red)
		middleRightRamp := ui.NewColorRamp(float64(width)/2+1, float64(width-1), red, theme.Background())

		for x := range width {
			var color twin.Color
			if x <= width/2 {
				color = leftMiddleRamp.AtInt(x)
			} else {
				color = middleRightRamp.AtInt(x)
			}
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
