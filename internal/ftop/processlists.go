package ftop

import "github.com/walles/ftop/internal/processes"

// Create a new list where ui.pickedProcess has been moved to index
// ui.pickedLine. This is a stable sort, so the order of all other processes is
// preserved.
func (ui *Ui) fixPickedProcess(procs []processes.Process) []processes.Process {
	if ui.pickedProcess == nil {
		return procs
	}

	// Find the current index
	currentIndex := -1
	for i, proc := range procs {
		if proc.SameAs(*ui.pickedProcess) {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		// Picked process is not in the list, probably because it has exited. Clear the pick.
		ui.pickedProcess = nil
		ui.pickedLine = nil
		return procs
	}

	if currentIndex == *ui.pickedLine {
		// Picked process is already in the right place
		return procs
	}

	reArranged := make([]processes.Process, 0, len(procs))
	targetIndex := *ui.pickedLine

	if currentIndex > targetIndex {
		// Moving earlier: Insert picked process at target, shift others right
		reArranged = append(reArranged, procs[0:targetIndex]...)
		reArranged = append(reArranged, procs[currentIndex])
		reArranged = append(reArranged, procs[targetIndex:currentIndex]...)
		reArranged = append(reArranged, procs[currentIndex+1:]...)
	} else {
		// Moving later: Remove picked process, shift others left, insert at target
		reArranged = append(reArranged, procs[0:currentIndex]...)
		reArranged = append(reArranged, procs[currentIndex+1:targetIndex+1]...)
		reArranged = append(reArranged, procs[currentIndex])
		reArranged = append(reArranged, procs[targetIndex+1:]...)
	}

	return reArranged
}
