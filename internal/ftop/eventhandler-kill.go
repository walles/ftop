package ftop

import (
	"os"
	"syscall"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/moor/v2/twin"
)

type eventHandlerKill struct {
	ui      *Ui
	process *processes.Process
}

func (h *eventHandlerKill) onRune(r rune) {
	defer func() {
		// After handling the kill, switch back to the base event handler
		h.ui.eventHandler = &eventHandlerBase{ui: h.ui}
	}()

	if r != 'k' {
		// Abort
		return
	}

	if h.ui.pickedProcess == nil {
		// Process not available
		log.Infof("No process selected, cannot kill")
		return
	}

	// Kill the process

	p, err := os.FindProcess(h.ui.pickedProcess.Pid)
	if err != nil {
		log.Infof("Process %s not found for killing: %v", h.ui.pickedProcess.String(), err)
		return
	}

	err = p.Signal(syscall.SIGKILL)
	if err != nil {
		log.Infof("Failed to kill process %s: %v", h.ui.pickedProcess.String(), err)
		return
	}

	log.Debugf("Killed process %s", h.ui.pickedProcess.String())

}

func (h *eventHandlerKill) onKeyCode(keyCode twin.KeyCode) {
	if keyCode == twin.KeyEscape {
		h.ui.eventHandler = &eventHandlerBase{ui: h.ui}
		return
	}
}
