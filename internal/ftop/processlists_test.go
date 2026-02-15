package ftop

import (
	"testing"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

// Helper to create a test process
func makeProcess(pid int, command string) processes.Process {
	return processes.Process{
		Pid:      pid,
		Command:  command,
		Username: "testuser",
		RssKb:    1000,
		CpuTime:  toDuration(100),
	}
}

// Helper to create a Ui with a mock screen
func makeTestUi() *Ui {
	screen := twin.NewFakeScreen(80, 24)
	return NewUi(screen, themes.NewTheme("auto", nil))
}

func TestFixPickedProcess_MoveEarlier(t *testing.T) {
	ui := makeTestUi()

	procs := []processes.Process{
		makeProcess(1, "one"),
		makeProcess(2, "two"),
		makeProcess(3, "three"),
		makeProcess(4, "four"),
		makeProcess(5, "five"),
	}

	// Pick process at index 3 (four), and want it at index 1
	ui.pickedProcess = &procs[3]
	targetLine := 1
	ui.pickedLine = &targetLine

	result := ui.fixPickedProcess(procs)

	// Expected order: one, four, two, three, five
	assert.Equal(t, len(result), 5)
	assert.Equal(t, result[0].Pid, 1)
	assert.Equal(t, result[1].Pid, 4) // Moved here
	assert.Equal(t, result[2].Pid, 2)
	assert.Equal(t, result[3].Pid, 3)
	assert.Equal(t, result[4].Pid, 5)
}

func TestFixPickedProcess_MoveLater(t *testing.T) {
	ui := makeTestUi()

	procs := []processes.Process{
		makeProcess(1, "one"),
		makeProcess(2, "two"),
		makeProcess(3, "three"),
		makeProcess(4, "four"),
		makeProcess(5, "five"),
	}

	// Pick process at index 1 (two), and want it at index 3
	ui.pickedProcess = &procs[1]
	targetLine := 3
	ui.pickedLine = &targetLine

	result := ui.fixPickedProcess(procs)

	// Expected order: one, three, four, two, five
	assert.Equal(t, len(result), 5)
	assert.Equal(t, result[0].Pid, 1)
	assert.Equal(t, result[1].Pid, 3)
	assert.Equal(t, result[2].Pid, 4)
	assert.Equal(t, result[3].Pid, 2) // Moved here
	assert.Equal(t, result[4].Pid, 5)
}

func TestFixPickedProcess_PickedProcessNotInList(t *testing.T) {
	ui := makeTestUi()

	procs := []processes.Process{
		makeProcess(1, "one"),
		makeProcess(2, "two"),
		makeProcess(3, "three"),
	}

	// Pick a process that's not in the list (simulating process exit)
	notInList := makeProcess(999, "missing")
	ui.pickedProcess = &notInList
	targetLine := 1
	ui.pickedLine = &targetLine

	result := ui.fixPickedProcess(procs)

	// Order should be unchanged
	assert.Equal(t, len(result), 3)
	assert.Equal(t, result[0].Pid, 1)
	assert.Equal(t, result[1].Pid, 2)
	assert.Equal(t, result[2].Pid, 3)

	// Pick should be cleared
	assert.Equal(t, ui.pickedProcess == nil, true)
	assert.Equal(t, ui.pickedLine == nil, true)
}

func TestFixPickedProcess_MoveToFirstPosition(t *testing.T) {
	ui := makeTestUi()

	procs := []processes.Process{
		makeProcess(1, "one"),
		makeProcess(2, "two"),
		makeProcess(3, "three"),
		makeProcess(4, "four"),
	}

	// Pick process at index 2 (three), move to index 0
	ui.pickedProcess = &procs[2]
	targetLine := 0
	ui.pickedLine = &targetLine

	result := ui.fixPickedProcess(procs)

	// Expected order: three, one, two, four
	assert.Equal(t, len(result), 4)
	assert.Equal(t, result[0].Pid, 3) // Moved here
	assert.Equal(t, result[1].Pid, 1)
	assert.Equal(t, result[2].Pid, 2)
	assert.Equal(t, result[3].Pid, 4)
}

func TestFixPickedProcess_MoveToLastPosition(t *testing.T) {
	ui := makeTestUi()

	procs := []processes.Process{
		makeProcess(1, "one"),
		makeProcess(2, "two"),
		makeProcess(3, "three"),
		makeProcess(4, "four"),
	}

	// Pick process at index 1 (two), move to last position (index 3)
	ui.pickedProcess = &procs[1]
	targetLine := 3
	ui.pickedLine = &targetLine

	result := ui.fixPickedProcess(procs)

	// Expected order: one, three, four, two
	assert.Equal(t, len(result), 4)
	assert.Equal(t, result[0].Pid, 1)
	assert.Equal(t, result[1].Pid, 3)
	assert.Equal(t, result[2].Pid, 4)
	assert.Equal(t, result[3].Pid, 2) // Moved here
}
