package ftop

import "github.com/walles/moor/v2/twin"

type eventHandlerFilter struct {
	ui *Ui
}

func (h *eventHandlerFilter) onRune(r rune) {
	// FIXME: Add to the filter string here
}

func (h *eventHandlerFilter) onKeyCode(keyCode twin.KeyCode) {
	if keyCode == twin.KeyEscape || keyCode == twin.KeyEnter {
		// Switch back to the default event handler
		h.ui.eventHandler = &eventHandlerBase{ui: h.ui}
	}
}
