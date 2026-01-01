package main

import (
	"fmt"
	"os"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/processes"
)

type processListUpdated struct{}

func main() {
	screen, err := twin.NewScreen()
	if err != nil {
		fmt.Println("Error creating screen:", err)
		os.Exit(1)
	}

	procsTracker, err := processes.NewTracker()
	if err != nil {
		fmt.Println("Error creating process tracker:", err)
		os.Exit(1)
	}

	events := make(chan twin.Event)
	go func() {
		for event := range screen.Events() {
			events <- event
		}
	}()
	go func() {
		for range procsTracker.OnUpdate {
			events <- processListUpdated{}
		}
	}()

	for {
		event := <-events

		if _, ok := event.(twin.EventResize); ok {
			allProcesses := procsTracker.GetProcesses()
			processes.Render(allProcesses, screen)
		}

		if _, ok := event.(processListUpdated); ok {
			allProcesses := procsTracker.GetProcesses()
			processes.Render(allProcesses, screen)
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

	screen.Close()
}
