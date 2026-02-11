package ftop

import "github.com/walles/moor/v2/twin"

type eventHandlerBase struct {
	ui *Ui
}

func (h *eventHandlerBase) onRune(r rune) {
	if r == 'q' {
		h.ui.done = true
		return
	}

	if r == '/' || r == 'f' {
		// Switch to the filter event handler
		h.ui.eventHandler = &eventHandlerFilter{ui: h.ui}
	}
}

func (h *eventHandlerBase) onKeyCode(keyCode twin.KeyCode) {
	if keyCode == twin.KeyEscape {
		h.ui.done = true
		return
	}
}
