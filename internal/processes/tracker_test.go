package processes

import (
	"testing"
	"time"

	"github.com/walles/ftop/internal/assert"
)

func TestPreserveDyingProcessCommands(t *testing.T) {
	// Create a start time for our test processes
	startTime := time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)
	differentStartTime := startTime.Add(SAME_PROCESS_STARTTIME_TOLERANCE + time.Millisecond)

	t.Run("preserves command when process is dying", func(t *testing.T) {
		// Previous frame: healthy bash process running hello.sh
		previous := map[int]*Process{
			1234: {
				Pid:       1234,
				startTime: startTime,
				Cmdline:   "bash hello.sh",
			},
		}

		// Current frame: same process but now dying (parenthesized)
		current := []*Process{
			{
				Pid:       1234,
				startTime: startTime,
				Cmdline:   "(bash)",
			},
		}

		currentByPid := map[int]*Process{current[0].Pid: current[0]}
		matches, _ := buildProcessMatches(previous, currentByPid)

		// Apply preservation logic
		preserveDyingProcessCommands(matches)

		// Verify command was preserved
		assert.Equal(t, "bash hello.sh", current[0].Cmdline)
		assert.Equal(t, "hello.sh", current[0].Command())
	})

	// From "man ps" on macOS
	t.Run("preserves command from <defunct> processes", func(t *testing.T) {
		// Previous frame: healthy bash process running hello.sh
		previous := map[int]*Process{
			1234: {
				Pid:       1234,
				startTime: startTime,
				Cmdline:   "bash hello.sh",
			},
		}

		// Current frame: same process but now defunct
		current := []*Process{
			{
				Pid:       1234,
				startTime: startTime,
				Cmdline:   "<defunct>",
			},
		}

		currentByPid := map[int]*Process{current[0].Pid: current[0]}
		matches, _ := buildProcessMatches(previous, currentByPid)

		// Apply preservation logic
		preserveDyingProcessCommands(matches)

		// Verify command was preserved
		assert.Equal(t, current[0].Cmdline, "bash hello.sh")
		assert.Equal(t, current[0].Command(), "hello.sh")
	})

	// From "man ps" on macOS
	t.Run("preserves command from <exiting> processes", func(t *testing.T) {
		// Previous frame: healthy bash process running hello.sh
		previous := map[int]*Process{
			1234: {
				Pid:       1234,
				startTime: startTime,
				Cmdline:   "bash hello.sh",
			},
		}

		// Current frame: same process but now exiting
		current := []*Process{
			{
				Pid:       1234,
				startTime: startTime,
				Cmdline:   "<exiting>",
			},
		}

		currentByPid := map[int]*Process{current[0].Pid: current[0]}
		matches, _ := buildProcessMatches(previous, currentByPid)

		// Apply preservation logic
		preserveDyingProcessCommands(matches)

		// Verify command was preserved
		assert.Equal(t, current[0].Cmdline, "bash hello.sh")
		assert.Equal(t, current[0].Command(), "hello.sh")
	})

	t.Run("does not preserve when PID is reused with different start time", func(t *testing.T) {
		// Previous frame: bash process
		previous := map[int]*Process{
			1234: {
				Pid:       1234,
				startTime: startTime,
				Cmdline:   "bash hello.sh",
			},
		}

		// Current frame: different process with same PID but start time outside
		// SameAs tolerance
		current := []*Process{
			{
				Pid:       1234,
				startTime: differentStartTime,
				Cmdline:   "(python)",
			},
		}

		currentByPid := map[int]*Process{current[0].Pid: current[0]}
		matches, _ := buildProcessMatches(previous, currentByPid)

		// Apply preservation logic
		preserveDyingProcessCommands(matches)

		// Verify command was NOT preserved (different process)
		assert.Equal(t, "(python)", current[0].Cmdline)
		assert.Equal(t, "(python)", current[0].Command())
	})

	t.Run("does not modify non-parenthesized processes", func(t *testing.T) {
		// Previous frame
		previous := map[int]*Process{
			1234: {
				Pid:       1234,
				startTime: startTime,
				Cmdline:   "bash hello.sh",
			},
		}

		// Current frame: process still alive (not parenthesized)
		current := []*Process{
			{
				Pid:       1234,
				startTime: startTime,
				Cmdline:   "bash goodbye.sh", // Changed its command
			},
		}

		currentByPid := map[int]*Process{current[0].Pid: current[0]}
		matches, _ := buildProcessMatches(previous, currentByPid)

		// Apply preservation logic
		preserveDyingProcessCommands(matches)

		// Verify command was NOT modified (not dying, so no preservation)
		assert.Equal(t, "bash goodbye.sh", current[0].Cmdline)
		assert.Equal(t, "goodbye.sh", current[0].Command())
	})

	t.Run("handles nil previous map", func(t *testing.T) {
		// Current frame: dying process
		current := []*Process{
			{
				Pid:       1234,
				startTime: startTime,
				Cmdline:   "(bash)",
			},
		}

		// Should not panic
		preserveDyingProcessCommands(ProcessMatching{})

		// Verify nothing changed
		assert.Equal(t, "(bash)", current[0].Cmdline)
	})

	t.Run("handles process not in previous frame", func(t *testing.T) {
		// Previous frame: different process
		previous := map[int]*Process{
			5678: {
				Pid:       5678,
				startTime: startTime,
				Cmdline:   "bash hello.sh",
			},
		}

		// Current frame: new dying process not in previous
		current := []*Process{
			{
				Pid:       1234,
				startTime: startTime,
				Cmdline:   "(bash)",
			},
		}

		currentByPid := map[int]*Process{current[0].Pid: current[0]}
		matches, _ := buildProcessMatches(previous, currentByPid)

		// Apply preservation logic
		preserveDyingProcessCommands(matches)

		// Verify nothing changed (process wasn't in previous frame)
		assert.Equal(t, "(bash)", current[0].Cmdline)
	})
}
