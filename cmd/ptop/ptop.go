package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/io"
	"github.com/walles/ptop/internal/log"
	"github.com/walles/ptop/internal/processes"
	"github.com/walles/ptop/internal/ptop"
)

type processListUpdated struct{}

func main() {
	os.Exit(internalMain())
}

// Never call os.Exit() from inside of this function because that will cause us
// not to shut down the screen properly.
//
// Returns the program's exit code.
//
// Example:
//
//	func main() {
//	    os.Exit(internalMain())
//	}
func internalMain() int {
	screen, err := twin.NewScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating screen:", err)
		return 1
	}

	defer onExit(screen)

	defer func() {
		log.PanicHandler("main", recover(), debug.Stack())
	}()

	procsTracker := processes.NewTracker()
	ioTracker := io.NewTracker()

	events := make(chan twin.Event)
	go func() {
		defer func() {
			log.PanicHandler("main/screen events poller", recover(), debug.Stack())
		}()
		for event := range screen.Events() {
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

	for {
		event := <-events

		if _, ok := event.(twin.EventResize); ok {
			allProcesses := procsTracker.Processes()
			ptop.Render(allProcesses, ioTracker.Stats(), screen)
		}

		if _, ok := event.(processListUpdated); ok {
			allProcesses := procsTracker.Processes()
			ptop.Render(allProcesses, ioTracker.Stats(), screen)
		}

		if event, ok := event.(twin.EventRune); ok {
			if event.Rune() == 'q' {
				break
			}
		}

		if event, ok := event.(twin.EventKeyCode); ok {
			if event.KeyCode() == twin.KeyEscape {
				break
			}
		}
	}

	return 0
}

func onExit(screen twin.Screen) {
	screen.Close()

	if !log.HasErrors() {
		return
	}

	// FIXME: Print error reporting instructions with the log output

	fmt.Fprint(os.Stderr, log.String(true))

	os.Exit(1)
}
