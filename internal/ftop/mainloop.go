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

	// With race detection enabled (makes everything slow) and holding the down
	// arrow key, I saw event queues of at most 3. 10 will give us some headroom
	// on top of that.
	events := make(chan twin.Event, 10)
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

		if len(events) > 0 {
			// More events to handle, don't redraw until the queue is empty.
			continue
		}

		procs := procsTracker.Processes()
		procs = processes.Filter(procs, ui.filter)
		ui.Render(procs, ioTracker.Stats(), procsTracker.Launches())
	}
}
