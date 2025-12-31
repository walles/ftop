package main

import (
	"fmt"
	"os"

	"github.com/walles/moor/v2/twin"
	"github.com/walles/ptop/internal/processes"
)

func main() {
	allProcesses, err := processes.GetAll()
	if err != nil {
		fmt.Println("Error retrieving processes:", err)
		os.Exit(1)
	}

	screen, err := twin.NewScreen()
	if err != nil {
		fmt.Println("Error creating screen:", err)
		os.Exit(1)
	}

	processes.Render(processes.ByCpuPercent(allProcesses), screen)

	for {
		event := <-screen.Events()

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
