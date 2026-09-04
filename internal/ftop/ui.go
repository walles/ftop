package ftop

import (
	"strconv"
	"sync/atomic"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/ftop/internal/themes"
	"github.com/walles/twin"
)

type eventHandler interface {
	onRune(r rune)
	onKeyCode(keyCode twin.KeyCode)
}

type Ui struct {
	theme  themes.Theme
	screen twin.Screen

	eventHandler eventHandler
	events       chan any

	filter string // Empty means no filter

	// PID the command line named, if it named one. Taken by
	// takeInitialPageProcess() and nil ever after.
	initialPagePid *int

	// Atomic because RequestShutdown() sets it from whichever goroutine is
	// asking us to stop, while MainLoop() reads it.
	done atomic.Bool

	// nil means no line picked. If the value is too large it should be updated
	// by the rendering code.
	pickedLine *int

	// This will be updated during rendering
	pickedProcess *processes.Process

	// Unfiltered process list as of the most recent frame. The process info
	// page needs to look beyond whatever the user is filtering on.
	allProcesses []processes.Process

	// At this width or wider, we have always managed to render all three panes.
	// Below this, we shouldn't even try.
	//
	// The point is to avoid flipflopping between one and three panes, that's
	// annoying to look at.
	minThreePanesScreenWidth int
}

func NewUi(screen twin.Screen, theme themes.Theme, initialFilter string) *Ui {
	ui := &Ui{
		theme:  theme,
		screen: screen,
		filter: initialFilter,

		// With race detection enabled (makes everything slow) and holding the down
		// arrow key, I saw event queues of at most 3. 10 will give us some headroom
		// on top of that.
		events: make(chan any, 10),

		minThreePanesScreenWidth: 0, // Will be kept up to date by ftop.Ui.Render()
	}

	ui.eventHandler = &eventHandlerBase{ui: ui}

	// Somebody filtering on a number is asking about a process, so if it turns
	// out to be a live PID we'll show them that process right away. See
	// takeInitialPageProcess().
	//
	// Round tripped through strconv so that we only claim spellings the filter
	// can match too: processes.Process.Matches() looks for the filter inside a
	// plain Itoa() of the PID, where neither "007" nor "+7" can be found.
	if pid, err := strconv.Atoi(initialFilter); err == nil && strconv.Itoa(pid) == initialFilter {
		ui.initialPagePid = &pid
	}

	return ui
}

// The process the command line asked us to show a process info page for, or nil
// if there is none.
//
// Returns non-nil at most once, and only while there is still a startup PID to
// look up, so this is cheap to call once per frame.
//
// An empty process list means the tracker hasn't reported in yet, so the PID
// stays pending. Call this again next frame.
func (ui *Ui) takeInitialPageProcess() *processes.Process {
	if ui.initialPagePid == nil {
		return nil
	}

	if len(ui.allProcesses) == 0 {
		return nil
	}

	pid := *ui.initialPagePid
	ui.initialPagePid = nil

	for i := range ui.allProcesses {
		if ui.allProcesses[i].Pid != pid {
			continue
		}

		return &ui.allProcesses[i]
	}

	log.Infof("No process with PID %d, showing it as a filter only", pid)
	return nil
}

// Select proc, on the top line of the interactive process list.
//
// The selection outranks the sort order: as long as proc is alive and matches
// the filter, the list rendering keeps it on the top line.
func (ui *Ui) pickProcessAtTop(proc *processes.Process) {
	topLine := 0
	ui.pickedLine = &topLine
	ui.pickedProcess = proc
}
