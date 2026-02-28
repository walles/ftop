package ftop

import "github.com/walles/ftop/internal/processes"

func sortProcessesForDisplay(processesRaw []processes.Process) []processes.Process {
	return SortByScore(processesRaw, func(p processes.Process) stats {
		return stats{
			// The name in this case is really a fallback sort key for when the
			// other sort keys are all equal.
			name:     p.String(),
			cpuTime:  p.CpuTimeOrZero(),
			rssKb:    p.RssKb,
			nativity: p.Nativity,
		}
	})
}

// syncPickedProcess keeps the current pick coherent between picked line and
// picked process for this frame.
//
// It resolves pickedLine into pickedProcess, keeps the same process selected
// when possible, and optionally clamps selection to visible rows. If
// maxVisibleProcessIndex is below zero, visible-row clamping is skipped.
func (ui *Ui) syncPickedProcess(processesRaw []processes.Process, maxVisibleProcessIndex int) {
	if ui.pickedLine == nil {
		ui.pickedProcess = nil
		return
	}

	if *ui.pickedLine < 0 {
		ui.pickedLine = nil
		ui.pickedProcess = nil
		return
	}

	processesByScore := sortProcessesForDisplay(processesRaw)
	if len(processesByScore) == 0 {
		ui.pickedLine = nil
		ui.pickedProcess = nil
		return
	}

	maxPickableIndex := len(processesByScore) - 1
	if maxVisibleProcessIndex >= 0 && maxVisibleProcessIndex < maxPickableIndex {
		maxPickableIndex = maxVisibleProcessIndex
	}

	if maxPickableIndex < 0 {
		ui.pickedLine = nil
		ui.pickedProcess = nil
		return
	}

	if *ui.pickedLine > maxPickableIndex {
		// Keep pickedProcess unchanged here: if set, fixPickedProcess() will
		// keep that same process selected and move it to the clamped row.
		ui.pickedLine = &maxPickableIndex
	}

	if ui.pickedProcess != nil {
		processesByScore = ui.fixPickedProcess(processesByScore)

		if ui.pickedLine == nil {
			ui.pickedProcess = nil
			return
		}
	}

	ui.pickedProcess = &processesByScore[*ui.pickedLine]
}

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
