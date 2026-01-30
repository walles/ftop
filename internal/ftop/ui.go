package ftop

import (
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

type Ui struct {
	theme  themes.Theme
	screen twin.Screen

	// At this width or wider, we have always managed to render all three panes.
	// Below this, we shouldn't even try.
	//
	// The point is to avoid flipflopping between one and three panes, that's
	// annoying to look at.
	minThreePanesScreenWidth int
}

func NewUi(screen twin.Screen, theme themes.Theme) *Ui {
	return &Ui{
		theme:                    theme,
		screen:                   screen,
		minThreePanesScreenWidth: 0, // Will be kept up to date by ftop.Ui.Render()
	}
}
