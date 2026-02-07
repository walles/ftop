package processes

import (
	"testing"
	"time"
)

// TestCommandNameChange reproduces the crash when a process changes its command
// name between polling intervals. This happens when the same PID with the same
// start time appears with different command names in consecutive GetAll() calls.
func TestCommandNameChange(t *testing.T) {
	d := deduplicator{}

	// Start time for our test process
	startTime := time.Date(2026, 2, 7, 18, 56, 40, 0, time.UTC)
	otherStartTime := time.Date(2026, 2, 7, 18, 56, 39, 0, time.UTC)

	// First iteration: Process appears with command "foo"
	proc1 := &Process{
		Pid:       47070,
		startTime: startTime,
		Command:   "foo",
	}
	d.register(proc1)

	// Register TWO other processes with command "git" so there's already multiple in byName["git"]
	// This will force the disambiguator to loop through the list and expose the panic
	otherGit1 := &Process{
		Pid:       99998,
		startTime: otherStartTime,
		Command:   "git",
	}
	d.register(otherGit1)

	otherGit2 := &Process{
		Pid:       99999,
		startTime: time.Date(2026, 2, 7, 18, 56, 38, 0, time.UTC),
		Command:   "git",
	}
	d.register(otherGit2)

	// Verify first process can be disambiguated with command "foo"
	result := d.disambiguator(proc1)
	if result != "" {
		t.Errorf("Expected empty disambiguator for single process, got %q", result)
	}

	// Second iteration: Same process (same PID, same start time) but different command name
	// This simulates what happens when git or another process changes its command line
	proc2 := &Process{
		Pid:       47070,
		startTime: startTime,
		Command:   "git", // Different command name
	}
	d.register(proc2)

	// This should not panic - the process should be found in d.byName["git"]
	// Currently this will panic because proc2 is not in d.byName["git"],
	// it's still only in d.byName["foo"], so the disambiguator loop won't find it
	//
	// When fixed, proc2 should get disambiguator "3" because it's the newest of 3 git processes:
	// - otherGit2 (startTime=38) -> "1"
	// - otherGit1 (startTime=39) -> "2"
	// - proc2 (startTime=40) -> "3"
	result = d.disambiguator(proc2)
	if result != "3" {
		t.Errorf("Expected disambiguator '3' for newest git process, got %q", result)
	}
}

// TestCommandNameChangeWithMultipleProcesses tests the scenario where there are
// multiple processes with the same command name, and one of them changes its name
func TestCommandNameChangeWithMultipleProcesses(t *testing.T) {
	d := deduplicator{}

	startTime1 := time.Date(2026, 2, 7, 18, 56, 40, 0, time.UTC)
	startTime2 := time.Date(2026, 2, 7, 18, 56, 41, 0, time.UTC)

	// Register two processes with command "git"
	proc1 := &Process{
		Pid:       47070,
		startTime: startTime1,
		Command:   "git",
	}
	proc2 := &Process{
		Pid:       47071,
		startTime: startTime2,
		Command:   "git",
	}
	d.register(proc1)
	d.register(proc2)

	// Both should have disambiguators since there are two "git" processes
	result1 := d.disambiguator(proc1)
	if result1 != "1" {
		t.Errorf("Expected disambiguator '1' for first git process, got %q", result1)
	}
	result2 := d.disambiguator(proc2)
	if result2 != "2" {
		t.Errorf("Expected disambiguator '2' for second git process, got %q", result2)
	}

	// Now proc1 changes its command to "ssh"
	proc1Updated := &Process{
		Pid:       47070,
		startTime: startTime1,
		Command:   "ssh",
	}
	d.register(proc1Updated)

	// proc1Updated should now have no disambiguator (it's the only "ssh")
	result := d.disambiguator(proc1Updated)
	if result != "" {
		t.Errorf("Expected empty disambiguator for single ssh process, got %q", result)
	}

	// let proc2 keep its disambiguator so users can still recognize it, even
	// though it is now the only "git" process
	result2 = d.disambiguator(proc2)
	if result2 != "2" {
		t.Errorf("Expected disambiguator '2' for single remaining git process after other changed, got %q", result2)
	}
}
