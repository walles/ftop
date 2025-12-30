package main

import (
	"fmt"

	"github.com/walles/ptop/internal/processes"
)

func main() {
	allProcesses, err := processes.GetAll()
	if err != nil {
		fmt.Println("Error retrieving processes:", err)
		return
	}

	for _, process := range allProcesses {
		fmt.Println(process.String())
	}
}
