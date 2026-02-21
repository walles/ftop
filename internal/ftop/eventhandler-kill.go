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
	ui         *Ui
	process    *processes.Process
	excuse     string
	lastSignal *syscall.Signal
}

// Returns an explanation if the kill failed, or the empty string if it succeeded
func (h *eventHandlerKill) kill(signal syscall.Signal) string {
	if h.ui.pickedProcess == nil {
		// Process not available
		return "No process selected, cannot kill"
	}

	// Kill the process

	p, err := os.FindProcess(h.process.Pid)
	if err != nil {
		return fmt.Sprintf("Not found for killing: %v", err)
	}

	err = p.Signal(signal)
	if err != nil {
		return err.Error()
	}

	// Remember what we just did
	h.lastSignal = &signal

	log.Debugf("Killed process %s", h.ui.pickedProcess.String())
	return ""
}

func (h *eventHandlerKill) onRune(r rune) {
	if h.excuse != "" {
		// Kill was attempted but failed, user should have been informed, exit
		// on any key
		h.ui.eventHandler = &eventHandlerBase{ui: h.ui}
		return
	}

	if h.lastSignal != nil {
		// We have already started signalling, ignore all keyboard input except
		// for ESC, but that's handled in onKeyCode().
		return
	}

	if r != 'k' {
		// Abort
		h.ui.eventHandler = &eventHandlerBase{ui: h.ui}
		return
	}

	excuse := h.kill(syscall.SIGTERM)
	if excuse != "" {
		// Kill failed, set an excuse that we can show to the user
		h.excuse = excuse
	}
}

func (h *eventHandlerKill) onKeyCode(keyCode twin.KeyCode) {
	if keyCode == twin.KeyEscape {
		// FIXME: Do we need to stop some background process here from counting
		// down and sending signals?
		h.ui.eventHandler = &eventHandlerBase{ui: h.ui}
		return
	}

	if h.excuse != "" {
		// Kill was attempted but failed, exit on any key
		h.ui.eventHandler = &eventHandlerBase{ui: h.ui}
		return
	}
}
