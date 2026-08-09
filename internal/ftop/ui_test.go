package ftop

import (
	"testing"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/moor/v2/twin"
)

func TestNewUi_SetsInitialFilter(t *testing.T) {
	screen := twin.NewFakeScreen(80, 24)

	ui := NewUi(screen, themes.NewTheme("auto", nil), "firefox")

	assert.Equal(t, ui.filter, "firefox")
}

func TestNewUi_WithoutInitialFilter(t *testing.T) {
	screen := twin.NewFakeScreen(80, 24)

	ui := NewUi(screen, themes.NewTheme("auto", nil), "")

	assert.Equal(t, ui.filter, "")
}

// A PID on the command line means "show me this process", so the process info
// page for it comes up as soon as we know about that process.
func TestTakeInitialPageProcess(t *testing.T) {
	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "7619")
	ui.allProcesses = []processes.Process{{Pid: 1234}, {Pid: 7619}}

	proc := ui.takeInitialPageProcess()

	assert.Equal(t, proc, &ui.allProcesses[1])
}

// The main loop asks once per frame, but the command line only asked for one
// page.
func TestTakeInitialPageProcess_OnlyOnce(t *testing.T) {
	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "7619")
	ui.allProcesses = []processes.Process{{Pid: 7619}}
	ui.takeInitialPageProcess()

	assert.Equal(t, ui.takeInitialPageProcess(), (*processes.Process)(nil))
}

// The process list arrives a moment after startup, so an empty one means "not
// yet" rather than "no such process".
func TestTakeInitialPageProcess_WaitsForTheProcessList(t *testing.T) {
	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "7619")

	assert.Equal(t, ui.takeInitialPageProcess(), (*processes.Process)(nil))

	ui.allProcesses = []processes.Process{{Pid: 7619}}
	assert.Equal(t, ui.takeInitialPageProcess(), &ui.allProcesses[0])
}

// With no process to page, the filter is all the user asked for.
func TestTakeInitialPageProcess_NoSuchPid(t *testing.T) {
	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "7619")
	ui.allProcesses = []processes.Process{{Pid: 1234}}

	assert.Equal(t, ui.takeInitialPageProcess(), (*processes.Process)(nil))
}

// Filtering on a command name shouldn't page anything.
func TestTakeInitialPageProcess_FilterIsNotAPid(t *testing.T) {
	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "firefox")
	ui.allProcesses = []processes.Process{{Pid: 1234}}

	assert.Equal(t, ui.takeInitialPageProcess(), (*processes.Process)(nil))
}

// Only spellings the filter can match count as PIDs, so that paging and
// filtering always agree on which process the user meant.
func TestTakeInitialPageProcess_PaddedNumberIsJustAFilter(t *testing.T) {
	ui := NewUi(twin.NewFakeScreen(80, 24), themes.NewTheme("auto", nil), "007")
	ui.allProcesses = []processes.Process{{Pid: 7}}

	assert.Equal(t, ui.takeInitialPageProcess(), (*processes.Process)(nil))
	assert.Equal(t, ui.allProcesses[0].Matches(ui.filter), false)
}

// Leaving the process info page lands the user on that process, on the top
// line of the interactive list, no matter where the sort order would otherwise
// have put it.
func TestPickProcessAtTop(t *testing.T) {
	ui := makeTestUi()
	procs := []processes.Process{
		{Pid: 1234, Cmdline: "busy", CpuTime: toDuration(100)},
		{Pid: 7619, Cmdline: "idle", CpuTime: toDuration(1)},
	}

	ui.pickProcessAtTop(&procs[1])
	ui.syncPickedProcess(procs, -1)

	assert.Equal(t, *ui.pickedLine, 0)
	assert.Equal(t, ui.pickedProcess.Pid, 7619)
}
