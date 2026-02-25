package processes

import (
	"testing"
	"time"

	"github.com/walles/ftop/internal/assert"
)

func TestPreserveDyingProcessCommands(t *testing.T) {
	// Create a start time for our test processes
	startTime := time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)
	differentStartTime := time.Date(2026, 2, 18, 10, 0, 1, 0, time.UTC)

	t.Run("preserves command when process is dying", func(t *testing.T) {
		// Previous frame: healthy bash process running hello.sh
		previous := map[int]*Process{
			1234: {
				Pid:              1234,
				startTime:        startTime,
				cmdline:          "bash hello.sh",
				Command:          "hello.sh",
				lowercaseCommand: "hello.sh",
			},
		}

		// Current frame: same process but now dying (parenthesized)
		current := []*Process{
			{
				Pid:              1234,
				startTime:        startTime,
				cmdline:          "(bash)",
				Command:          "(bash)",
				lowercaseCommand: "(bash)",
			},
		}

		// Apply preservation logic
		preserveDyingProcessCommands(current, previous)

		// Verify command was preserved
		assert.Equal(t, "bash hello.sh", current[0].cmdline)
		assert.Equal(t, "hello.sh", current[0].Command)
		assert.Equal(t, "hello.sh", current[0].lowercaseCommand)
	})

	// From "man ps" on macOS
	t.Run("preserves command from <defunct> processes", func(t *testing.T) {
		// Previous frame: healthy bash process running hello.sh
		previous := map[int]*Process{
			1234: {
				Pid:              1234,
				startTime:        startTime,
				cmdline:          "bash hello.sh",
				Command:          "hello.sh",
				lowercaseCommand: "hello.sh",
			},
		}

		// Current frame: same process but now defunct
		current := []*Process{
			{
				Pid:              1234,
				startTime:        startTime,
				cmdline:          "<defunct>",
				Command:          "<defunct>",
				lowercaseCommand: "<defunct>",
			},
		}

		// Apply preservation logic
		preserveDyingProcessCommands(current, previous)

		// Verify command was preserved
		assert.Equal(t, current[0].cmdline, "bash hello.sh")
		assert.Equal(t, current[0].Command, "hello.sh")
		assert.Equal(t, current[0].lowercaseCommand, "hello.sh")
	})

	// From "man ps" on macOS
	t.Run("preserves command from <exiting> processes", func(t *testing.T) {
		// Previous frame: healthy bash process running hello.sh
		previous := map[int]*Process{
			1234: {
				Pid:              1234,
				startTime:        startTime,
				cmdline:          "bash hello.sh",
				Command:          "hello.sh",
				lowercaseCommand: "hello.sh",
			},
		}

		// Current frame: same process but now exiting
		current := []*Process{
			{
				Pid:              1234,
				startTime:        startTime,
				cmdline:          "<exiting>",
				Command:          "<exiting>",
				lowercaseCommand: "<exiting>",
			},
		}

		// Apply preservation logic
		preserveDyingProcessCommands(current, previous)

		// Verify command was preserved
		assert.Equal(t, current[0].cmdline, "bash hello.sh")
		assert.Equal(t, current[0].Command, "hello.sh")
		assert.Equal(t, current[0].lowercaseCommand, "hello.sh")
	})

	t.Run("does not preserve when PID is reused with different start time", func(t *testing.T) {
		// Previous frame: bash process
		previous := map[int]*Process{
			1234: {
				Pid:              1234,
				startTime:        startTime,
				cmdline:          "bash hello.sh",
				Command:          "hello.sh",
				lowercaseCommand: "hello.sh",
			},
		}

		// Current frame: different process with same PID but different start time
		current := []*Process{
			{
				Pid:              1234,
				startTime:        differentStartTime, // Different start time = different process
				cmdline:          "(python)",
				Command:          "(python)",
				lowercaseCommand: "(python)",
			},
		}

		// Apply preservation logic
		preserveDyingProcessCommands(current, previous)

		// Verify command was NOT preserved (different process)
		assert.Equal(t, "(python)", current[0].cmdline)
		assert.Equal(t, "(python)", current[0].Command)
		assert.Equal(t, "(python)", current[0].lowercaseCommand)
	})

	t.Run("does not modify non-parenthesized processes", func(t *testing.T) {
		// Previous frame
		previous := map[int]*Process{
			1234: {
				Pid:              1234,
				startTime:        startTime,
				cmdline:          "bash hello.sh",
				Command:          "hello.sh",
				lowercaseCommand: "hello.sh",
			},
		}

		// Current frame: process still alive (not parenthesized)
		current := []*Process{
			{
				Pid:              1234,
				startTime:        startTime,
				cmdline:          "bash goodbye.sh", // Changed its command
				Command:          "goodbye.sh",
				lowercaseCommand: "goodbye.sh",
			},
		}

		// Apply preservation logic
		preserveDyingProcessCommands(current, previous)

		// Verify command was NOT modified (not dying, so no preservation)
		assert.Equal(t, "bash goodbye.sh", current[0].cmdline)
		assert.Equal(t, "goodbye.sh", current[0].Command)
		assert.Equal(t, "goodbye.sh", current[0].lowercaseCommand)
	})

	t.Run("handles nil previous map", func(t *testing.T) {
		// No previous frame
		var previous map[int]*Process = nil

		// Current frame: dying process
		current := []*Process{
			{
				Pid:              1234,
				startTime:        startTime,
				cmdline:          "(bash)",
				Command:          "(bash)",
				lowercaseCommand: "(bash)",
			},
		}

		// Should not panic
		preserveDyingProcessCommands(current, previous)

		// Verify nothing changed
		assert.Equal(t, "(bash)", current[0].cmdline)
	})

	t.Run("handles process not in previous frame", func(t *testing.T) {
		// Previous frame: different process
		previous := map[int]*Process{
			5678: {
				Pid:              5678,
				startTime:        startTime,
				cmdline:          "bash hello.sh",
				Command:          "hello.sh",
				lowercaseCommand: "hello.sh",
			},
		}

		// Current frame: new dying process not in previous
		current := []*Process{
			{
				Pid:              1234,
				startTime:        startTime,
				cmdline:          "(bash)",
				Command:          "(bash)",
				lowercaseCommand: "(bash)",
			},
		}

		// Apply preservation logic
		preserveDyingProcessCommands(current, previous)

		// Verify nothing changed (process wasn't in previous frame)
		assert.Equal(t, "(bash)", current[0].cmdline)
	})
}
