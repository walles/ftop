package ftop

import (
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

type eventHandler interface {
	onRune(r rune)
	onKeyCode(keyCode twin.KeyCode)
}

type Ui struct {
	theme  themes.Theme
	screen twin.Screen

	eventHandler eventHandler
	events       chan any

	filter string // Empty means no filter

	done bool

	// nil means no line picked. If the value is too large it should be updated
	// by the rendering code.
	pickedLine *int

	// This will be updated during rendering
	pickedProcess *processes.Process

	// At this width or wider, we have always managed to render all three panes.
	// Below this, we shouldn't even try.
	//
	// The point is to avoid flipflopping between one and three panes, that's
	// annoying to look at.
	minThreePanesScreenWidth int
}

func NewUi(screen twin.Screen, theme themes.Theme) *Ui {
	ui := &Ui{
		theme:  theme,
		screen: screen,

		// With race detection enabled (makes everything slow) and holding the down
		// arrow key, I saw event queues of at most 3. 10 will give us some headroom
		// on top of that.
		events: make(chan any, 10),

		minThreePanesScreenWidth: 0, // Will be kept up to date by ftop.Ui.Render()
	}

	ui.eventHandler = &eventHandlerBase{ui: ui}

	return ui
}
