package ftop

import "github.com/walles/moor/v2/twin"

type eventHandlerFilter struct {
	ui *Ui
}

func (h *eventHandlerFilter) onRune(r rune) {
	h.ui.filter += string(r)
}

func (h *eventHandlerFilter) onKeyCode(keyCode twin.KeyCode) {
	if keyCode == twin.KeyEscape || keyCode == twin.KeyEnter {
		// Switch back to the default event handler
		h.ui.eventHandler = &eventHandlerBase{ui: h.ui}
		return
	}

	if keyCode == twin.KeyBackspace {
		// Unicode friendly delete-last-character
		runes := []rune(h.ui.filter)
		if len(runes) > 0 {
			h.ui.filter = string(runes[:len(runes)-1])
		}
	}
}
