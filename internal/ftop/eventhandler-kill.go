package ftop

import (
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/moor/v2/twin"
)

type eventHandlerKill struct {
	ui      *Ui
	process *processes.Process
}

func (h *eventHandlerKill) onRune(r rune) {
	// FIXME
}

func (h *eventHandlerKill) onKeyCode(keyCode twin.KeyCode) {
	if keyCode == twin.KeyEscape {
		h.ui.eventHandler = &eventHandlerBase{ui: h.ui}
		return
	}
}
