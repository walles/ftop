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

	if keyCode == twin.KeyBackspace {
		// Clear the filter
		h.ui.filter = ""
		return
	}

	if keyCode == twin.KeyDown {
		if h.ui.selectedLine == nil {
			// No line selected, select the first one
			h.ui.selectedLine = new(int)
		} else {
			// Move selection down
			*h.ui.selectedLine++
		}
		return
	}

	if keyCode == twin.KeyUp && h.ui.selectedLine != nil {
		if *h.ui.selectedLine > 0 {
			// Move selection up
			*h.ui.selectedLine--
		} else {
			// And it's gone!!
			h.ui.selectedLine = nil
		}
	}
}
