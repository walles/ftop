package ftop

import (
	"runtime/debug"

	"github.com/walles/ftop/internal/io"
	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/moor/v2/twin"
)

func (ui *Ui) MainLoop() {
	type processListUpdated struct{}

	procsTracker := processes.NewTracker()
	ioTracker := io.NewTracker()

	events := make(chan twin.Event)
	go func() {
		defer func() {
			log.PanicHandler("main/screen events poller", recover(), debug.Stack())
		}()
		for event := range ui.screen.Events() {
			events <- event
		}
	}()
	go func() {
		defer func() {
			log.PanicHandler("main/processes tracker poller", recover(), debug.Stack())
		}()
		for range procsTracker.OnUpdate {
			events <- processListUpdated{}
		}
	}()

	for !ui.done {
		switch event := (<-events).(type) {
		case twin.EventRune:
			ui.eventHandler.onRune(event.Rune())

		case twin.EventKeyCode:
			ui.eventHandler.onKeyCode(event.KeyCode())
		}

		procs := procsTracker.Processes()
		procs = processes.Filter(procs, ui.filter)
		ui.Render(procs, ioTracker.Stats(), procsTracker.Launches())
	}
}
