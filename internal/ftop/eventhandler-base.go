package ftop

import (
	"github.com/walles/ftop/internal/log"
	"github.com/walles/moor/v2/twin"
)

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

	if r == 'k' && h.ui.pickedProcess != nil {
		h.ui.eventHandler = &eventHandlerKill{ui: h.ui, process: h.ui.pickedProcess}
	}

	proc := h.ui.pickedProcess
	if r == 'i' && proc != nil {
		err := h.ui.screen.PauseAndCall(func() error {
			return h.ui.pageProcessInfo(proc)
		})
		if err != nil {
			log.Infof("Failed to page %s info: %v", proc.String(), err)
		}
	}
}

func (h *eventHandlerBase) onKeyCode(keyCode twin.KeyCode) {
	if keyCode == twin.KeyEscape {
		if h.ui.pickedLine != nil {
			// Clear the pick
			h.ui.pickedLine = nil
			h.ui.pickedProcess = nil
			return
		}

		if h.ui.filter != "" {
			// Clear the filter
			h.ui.filter = ""
			return
		}

		h.ui.done = true
		return
	}

	if keyCode == twin.KeyBackspace {
		// Clear the filter
		h.ui.filter = ""
		return
	}

	if keyCode == twin.KeyDown {
		if h.ui.pickedLine == nil {
			// No line picked, pick the first one
			h.ui.pickedLine = new(int)
			h.ui.pickedProcess = nil
		} else {
			// Move pick down
			*h.ui.pickedLine++
			h.ui.pickedProcess = nil
		}
		return
	}

	if keyCode == twin.KeyUp && h.ui.pickedLine != nil {
		if *h.ui.pickedLine > 0 {
			// Move pick up
			*h.ui.pickedLine--
			h.ui.pickedProcess = nil
		} else {
			// And it's gone!!
			h.ui.pickedLine = nil
			h.ui.pickedProcess = nil
		}
	}

	proc := h.ui.pickedProcess
	if keyCode == twin.KeyEnter && proc != nil {
		err := h.ui.screen.PauseAndCall(func() error {
			return h.ui.pageProcessInfo(proc)
		})
		if err != nil {
			log.Infof("Failed to page %s info: %v", proc.String(), err)
		}
	}
}
