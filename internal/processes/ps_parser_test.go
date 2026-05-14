package processes

import (
	"testing"
	"time"

	"github.com/walles/ftop/internal/assert"
)

func BenchmarkPsLineToProcess(b *testing.B) {
	snapshotTime := time.Now()

	// Prepare 1000 identical lines based on typical ps output
	const numLines = 1000
	lines := make([]string, numLines)
	for i := range lines {
		lines[i] = " 974 973 588 00:00 501 0.0 0:00.00 0.0 /bin/sleep foo bar"
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for _, line := range lines {
			_, err := psLineToProcess(line, snapshotTime)
			if err != nil {
				b.Fatalf("Parse error: %v", err)
			}
		}
	}
}

func TestPsLineToProcess_HappyPathMacOS(t *testing.T) {
	line := " 974 973 588 00:00 501 0.0 0:00.00 0.0 /bin/sleep"
	snapshotTime := time.Date(2026, time.March, 15, 9, 55, 27, 0, time.Local)

	proc, err := psLineToProcess(line, snapshotTime)
	assert.Equal(t, err, nil)

	assert.Equal(t, proc.Pid, 974)
	assert.Equal(t, proc.ppid, 973)
	assert.Equal(t, proc.RssKb, 588)
	assert.Equal(t, proc.startTime, snapshotTime)
	assert.Equal(t, proc.Username, uidToUsername(501))
	assert.Equal(t, proc.Command(), "sleep")
	assert.Equal(t, proc.Cmdline, "/bin/sleep")

	assert.Equal(t, true, proc.cpuPercent != nil)
	assert.Equal(t, *proc.cpuPercent, 0.0)

	assert.Equal(t, true, proc.CpuTime != nil)
	assert.Equal(t, *proc.CpuTime, time.Duration(0))

	assert.Equal(t, true, proc.memoryPercent != nil)
	assert.Equal(t, *proc.memoryPercent, 0.0)
}

func TestPsLineToProcess_HappyPathLinux(t *testing.T) {
	line := "    1     0  3196       00:21     0  0.1 00:00:00  0.0 bash"
	snapshotTime := time.Date(2026, time.March, 15, 8, 51, 54, 0, time.Local)

	proc, err := psLineToProcess(line, snapshotTime)
	assert.Equal(t, err, nil)

	assert.Equal(t, proc.Pid, 1)
	assert.Equal(t, proc.ppid, 0)
	assert.Equal(t, proc.RssKb, 3196)
	assert.Equal(t, proc.startTime, snapshotTime.Add(-21*time.Second))
	assert.Equal(t, proc.Username, uidToUsername(0))
	assert.Equal(t, proc.Command(), "bash")
	assert.Equal(t, proc.Cmdline, "bash")

	assert.Equal(t, true, proc.cpuPercent != nil)
	assert.Equal(t, *proc.cpuPercent, 0.1)

	assert.Equal(t, true, proc.CpuTime != nil)
	assert.Equal(t, *proc.CpuTime, time.Duration(0))

	assert.Equal(t, true, proc.memoryPercent != nil)
	assert.Equal(t, *proc.memoryPercent, 0.0)
}

func TestPsLineToProcess_StableAcrossEtimeRefreshes(t *testing.T) {
	lineA := "    1     0  3196       00:21     0  0.1 00:00:00  0.0 bash"
	lineB := "    1     0  3196       00:22     0  0.1 00:00:00  0.0 bash"

	snapshotA := time.Date(2026, time.March, 15, 8, 51, 54, 900000000, time.Local)
	snapshotB := time.Date(2026, time.March, 15, 8, 51, 55, 100000000, time.Local)

	procA, err := psLineToProcess(lineA, snapshotA)
	assert.Equal(t, err, nil)

	procB, err := psLineToProcess(lineB, snapshotB)
	assert.Equal(t, err, nil)

	delta := procA.startTime.Sub(procB.startTime).Abs()
	assert.Equal(t, delta <= SAME_PROCESS_STARTTIME_TOLERANCE, true)
	assert.Equal(t, procA.SameAs(procB), true)
}

// Real world example from macOS ps
func TestPsLineToProcess_MalformedElapsedTime(t *testing.T) {
	line := "24381 48334   1024       00:-1   501   0.0   0:00.00  0.0 netstat -ib"
	snapshotTime := time.Date(2026, time.March, 15, 11, 22, 0, 0, time.Local)

	proc, err := psLineToProcess(line, snapshotTime)
	assert.Equal(t, err, nil)

	assert.Equal(t, proc.Pid, 24381)
	assert.Equal(t, proc.Command(), "netstat")
	assert.Equal(t, proc.startTime, snapshotTime.Add(time.Second))
}

// This odd input has been observed from macOS /bin/ps in the wild, where etime
// can temporarily look like "-14:-1" during sleep / darkwake cycles.
//
// We keep this test to ensure one malformed process does not break refreshes:
// it should parse, be kept, and get a future startTime that the UI can mark as
// uncertain instead of dropping the whole snapshot.
//
// Environment where this was observed:
//
//	macOS 15.7.4 (24G517)
//	"PROGRAM:ps  PROJECT:adv_cmds-235" from "what /bin/ps"
func TestPsLineToProcess_NegativeMinutesElapsedTime(t *testing.T) {
	line := "54898 1 7384 -14:-1 0 0.0 0:00.08 0.0 /usr/libexec/mdmclient daemon"
	snapshotTime := time.Date(2026, time.March, 15, 11, 22, 0, 0, time.Local)

	proc, err := psLineToProcess(line, snapshotTime)
	assert.Equal(t, err, nil)

	assert.Equal(t, proc.Pid, 54898)
	assert.Equal(t, proc.Command(), "mdmclient")
	assert.Equal(t, proc.startTime, snapshotTime.Add(14*time.Minute+time.Second))
}
