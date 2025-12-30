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

	<-screen.Events()

	screen.Close()
}
