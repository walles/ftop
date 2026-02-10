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
		event := <-events

		switch e := event.(type) {
		case twin.EventResize:
			allProcesses := procsTracker.Processes()
			ui.Render(allProcesses, ioTracker.Stats(), procsTracker.Launches())

		case processListUpdated:
			allProcesses := procsTracker.Processes()
			ui.Render(allProcesses, ioTracker.Stats(), procsTracker.Launches())

		case twin.EventRune:
			ui.eventHandler.onRune(e.Rune())

		case twin.EventKeyCode:
			ui.eventHandler.onKeyCode(e.KeyCode())
		}
	}
}
