package ftop

import (
	"runtime/debug"

	"github.com/walles/ftop/internal/io"
	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/moor/v2/twin"
)

type replaceEventHandler struct {
	// To prevent stale updates, old must match before replacing
	old eventHandler
	new eventHandler
}

func (ui *Ui) MainLoop() {
	type processListUpdated struct{}

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
			ui.events <- processListUpdated{}
		}
	}()

	for !ui.done {
		switch event := (<-ui.events).(type) {
		case processListUpdated:
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
