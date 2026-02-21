package ftop

import (
	"fmt"
	"os"
	"syscall"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/moor/v2/twin"
)

type eventHandlerKill struct {
	ui      *Ui
	process *processes.Process
	excuse  string
}

// Returns an explanation if the kill failed, or the empty string if it succeeded
func (h *eventHandlerKill) kill() string {
	if h.ui.pickedProcess == nil {
		// Process not available
		return "No process selected, cannot kill"
	}

	// Kill the process

	p, err := os.FindProcess(h.process.Pid)
	if err != nil {
		return fmt.Sprintf("Not found for killing: %v", err)
	}

	err = p.Signal(syscall.SIGKILL)
	if err != nil {
		return err.Error()
	}

	log.Debugf("Killed process %s", h.ui.pickedProcess.String())
	return ""
}

func (h *eventHandlerKill) onRune(r rune) {
	if h.excuse != "" {
		// Kill was attempted but failed, exit on any key
		h.ui.eventHandler = &eventHandlerBase{ui: h.ui}
		return
	}

	if r != 'k' {
		// Abort
		h.ui.eventHandler = &eventHandlerBase{ui: h.ui}
		return
	}

	excuse := h.kill()
	if excuse == "" {
		// Kill succeeded, we are done
		h.ui.eventHandler = &eventHandlerBase{ui: h.ui}
		return
	}

	// Kill failed, set an excuse that we can show to the user
	h.excuse = excuse
}

func (h *eventHandlerKill) onKeyCode(keyCode twin.KeyCode) {
	if keyCode == twin.KeyEscape {
		h.ui.eventHandler = &eventHandlerBase{ui: h.ui}
		return
	}

	if h.excuse != "" {
		// Kill was attempted but failed, exit on any key
		h.ui.eventHandler = &eventHandlerBase{ui: h.ui}
		return
	}
}
