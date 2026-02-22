package ftop

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/moor/v2/twin"
)

type eventHandlerKill struct {
	ui      *Ui
	process *processes.Process

	lock       sync.RWMutex // Protects excuse and lastSignal
	excuse     string
	lastSignal *syscall.Signal
}

func (killer *eventHandlerKill) getExcuse() string {
	killer.lock.RLock()
	defer killer.lock.RUnlock()

	return killer.excuse
}

func (killer *eventHandlerKill) setExcuse(excuse string) {
	killer.lock.Lock()
	defer killer.lock.Unlock()

	killer.excuse = excuse
}

func (killer *eventHandlerKill) hasLastSignal() bool {
	killer.lock.RLock()
	defer killer.lock.RUnlock()

	return killer.lastSignal != nil
}

func (killer *eventHandlerKill) setLastSignal(signal syscall.Signal) {
	killer.lock.Lock()
	defer killer.lock.Unlock()

	killer.lastSignal = &signal
}

// Returns an explanation if the kill failed, or the empty string if it succeeded
func (killer *eventHandlerKill) kill(signal syscall.Signal) string {
	// Kill the process

	p, err := os.FindProcess(killer.process.Pid)
	if err != nil {
		return fmt.Sprintf("Not found for killing: %v", err)
	}

	err = p.Signal(signal)
	if err != nil {
		return err.Error()
	}

	// Remember what we just did
	killer.setLastSignal(signal)

	log.Debugf("Killed process %s", killer.process.String())
	return ""
}

func (killer *eventHandlerKill) onRune(r rune) {
	if killer.getExcuse() != "" {
		// Kill was attempted but failed, user should have been informed, exit
		// on any key
		killer.ui.eventHandler = &eventHandlerBase{ui: killer.ui}
		return
	}

	if killer.hasLastSignal() {
		// We have already started signalling, ignore all keyboard input except
		// for ESC, but that's handled in onKeyCode().
		return
	}

	if r != 'k' {
		// Abort
		killer.ui.eventHandler = &eventHandlerBase{ui: killer.ui}
		return
	}

	excuse := killer.kill(syscall.SIGTERM)
	if excuse != "" {
		// Kill failed, set an excuse that we can show to the user
		killer.setExcuse(excuse)
	}

	go func() {
		// Wait 5s for the process to die
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if !killer.process.IsAlive() {
				// It's gone!
				killer.ui.events <- replaceEventHandler{old: killer, new: &eventHandlerBase{ui: killer.ui}}
				return
			}
			time.Sleep(100 * time.Millisecond)
		}

		// It's still there, try SIGKILL
		excuse := killer.kill(syscall.SIGKILL)
		if excuse != "" {
			// Kill failed, set an excuse that we can show to the user
			killer.setExcuse(excuse)
			return
		}

		// Wait 5s for the process to die
		deadline = time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if !killer.process.IsAlive() {
				// It's gone!
				killer.ui.events <- replaceEventHandler{old: killer, new: &eventHandlerBase{ui: killer.ui}}
				return
			}
			time.Sleep(100 * time.Millisecond)
		}

		// Tell the user we failed
		killer.setExcuse("Process is still alive after SIGKILL")
	}()
}

func (killer *eventHandlerKill) onKeyCode(keyCode twin.KeyCode) {
	if keyCode == twin.KeyEscape {
		// FIXME: Do we need to stop some background process here from counting
		// down and sending signals?
		killer.ui.eventHandler = &eventHandlerBase{ui: killer.ui}
		return
	}

	if killer.getExcuse() != "" {
		// Kill was attempted but failed, exit on any key
		killer.ui.eventHandler = &eventHandlerBase{ui: killer.ui}
		return
	}
}
