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

	for !ui.done {
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

		procs := procsTracker.Processes()
		procs = processes.Filter(procs, ui.filter)
		ui.Render(procs, ioTracker.Stats(), procsTracker.Launches())
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
