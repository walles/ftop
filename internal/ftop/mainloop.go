package ftop

import (
	"runtime/debug"

	"github.com/walles/ftop/internal/io"
	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/moor/v2/twin"
)

// Replace current event handle with another one. This is basically a mode
// switch.
type replaceEventHandler struct {
	// To prevent stale updates, old must match before replacing
	old eventHandler
	new eventHandler
}

type redrawUi struct{}

func (ui *Ui) MainLoop() {
	procsTracker := processes.NewTracker()
	ioTracker := io.NewTracker()

	go func() {
		defer func() {
			log.PanicHandler("main/screen events poller", recover(), debug.Stack())
		}()
		for event := range ui.screen.Events() {
			ui.events <- event
		}
	}()
	go func() {
		defer func() {
			log.PanicHandler("main/processes tracker poller", recover(), debug.Stack())
		}()
		for range procsTracker.OnUpdate {
			ui.events <- redrawUi{}
		}
	}()

	for !ui.done.Load() {
		switch event := (<-ui.events).(type) {
		case redrawUi:
			// This block intentionally left blank since process list update
			// events only exist to trigger a redraw.

		case replaceEventHandler:
			if event.new == nil {
				panic("replaceEventHandler with nil new value")
			}

			if event.old != nil && ui.eventHandler != event.old {
				log.Infof("Not setting new event handler because old one did not match")
				continue
			}

			ui.eventHandler = event.new

		case twin.EventRune:
			ui.eventHandler.onRune(event.Rune())

		case twin.EventKeyCode:
			ui.eventHandler.onKeyCode(event.KeyCode())
		}

		if len(ui.events) > 0 {
			// More events to handle, don't redraw until the queue is empty.
			continue
		}

		ui.allProcesses = procsTracker.Processes()

		// Before the first Render() rather than after it, so that a PID on the
		// command line takes the user straight into the pager. Coming back out
		// of it lands them in the interactive view, filtered on that PID.
		//
		// Not while shutting down: a goroutine crashing before we get here
		// should get the user an exit and a crash report, not a pager that
		// holds ftop open until they quit it.
		if initialProc := ui.takeInitialPageProcess(); initialProc != nil && !ui.done.Load() {
			ui.pageProcessInfo(initialProc)

			// Leaving the pager should feel like having arrowed down to that
			// process, so that killing it or looking at it again is one
			// keypress away.
			ui.pickProcessAtTop(initialProc)
		}

		procs := processes.Filter(ui.allProcesses, ui.filter)
		ui.Render(procs, ioTracker.Stats(), procsTracker.Launches())
	}
}

// Asks MainLoop() to return, callable from any goroutine.
//
// Returns immediately, without waiting for the main loop to notice. The main
// loop finishes whatever it is doing first, so if the process info pager is up
// (see pageProcessInfo()), ftop keeps running until the user leaves the pager.
func (ui *Ui) RequestShutdown() {
	ui.done.Store(true)

	// Best effort wake up call, in case the main loop is waiting for something
	// to happen. If the queue is full then the main loop has events to process
	// anyway, and it will check ui.done as soon as it gets to them.
	select {
	case ui.events <- redrawUi{}:
	default:
	}
}

// This will request a redraw of the UI
func (ui *Ui) requestRedraw() {
	select {
	case ui.events <- redrawUi{}:
	default:
		log.Infof("Failed to request UI redraw")
	}
}
