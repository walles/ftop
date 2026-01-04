package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/log"
	"github.com/walles/ptop/internal/processes"
	"github.com/walles/ptop/internal/ptop"
)

type processListUpdated struct{}

func main() {
	screen, err := twin.NewScreen()
	if err != nil {
		fmt.Println("Error creating screen:", err)
		os.Exit(1)
	}

	defer onExit(screen)

	defer func() {
		log.PanicHandler("main", recover(), debug.Stack())
	}()

	procsTracker, err := processes.NewTracker()
	if err != nil {
		fmt.Println("Error creating process tracker:", err)
		os.Exit(1)
	}

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
			allProcesses := procsTracker.GetProcesses()
			ptop.Render(allProcesses, screen)
		}

		if _, ok := event.(processListUpdated); ok {
			allProcesses := procsTracker.GetProcesses()
			ptop.Render(allProcesses, screen)
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
}

func onExit(screen twin.Screen) {
	screen.Close()

	if !log.HasErrors() {
		return
	}

	// FIXME: Print error reporting instructions with the log output

	fmt.Fprint(os.Stderr, log.String())

	os.Exit(1)
}
