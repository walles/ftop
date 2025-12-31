package main

import (
	"fmt"
	"os"
	"time"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/processes"
)

func main() {
	screen, err := twin.NewScreen()
	if err != nil {
		fmt.Println("Error creating screen:", err)
		os.Exit(1)
	}

	events := make(chan twin.Event)
	go func() {
		for event := range screen.Events() {
			events <- event
		}
	}()
	go func() {
		for tick := range time.NewTicker(time.Second).C {
			events <- tick
		}
	}()

	for {
		allProcesses, err := processes.GetAll()
		if err != nil {
			fmt.Println("Error retrieving processes:", err)
			os.Exit(1)
		}

		processes.Render(processes.ByCpuPercent(allProcesses), screen)

		event := <-events

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
