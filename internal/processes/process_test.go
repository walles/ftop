package processes

import (
	"os"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/util"
)

type testTimeInternal struct {
	wall uint64
	ext  int64
	loc  *time.Location
}

func withWallFromAndMonotonicFrom(wallSource, monotonicSource time.Time) time.Time {
	combined := *(*testTimeInternal)(unsafe.Pointer(&wallSource))
	combined.ext = (*(*testTimeInternal)(unsafe.Pointer(&monotonicSource))).ext

	return *(*time.Time)(unsafe.Pointer(&combined))
}

const TEN_MB = 10 * 1024 * 1024

func TestGetAll(t *testing.T) {
	procs, err := GetAll()
	assert.Equal(t, err, nil)
	assert.Equal(t, true, len(procs) > 0)

	var self *Process
	for _, proc := range procs {
		if proc.Pid == os.Getpid() {
			self = proc
			break
		}
	}
	assert.Equal(t, false, self == nil)

	// Validate Pid field
	assert.Equal(t, os.Getpid(), self.Pid)

	// Validate ppid field
	assert.Equal(t, os.Getppid(), self.ppid)

	assert.Equal(t, true, self.Command() == "processes.test" || strings.Contains(self.Command(), "debug"))
	assert.Equal(t, self.lowercaseCommand, strings.ToLower(self.Command()))

	// Validate Username field
	assert.Equal(t, self.Username, util.GetCurrentUsername())

	// Validate RssKb field
	assert.Equal(t, true, self.RssKb > 0)

	// If this is not enough, feel free to increase it. But it should be low
	// enough to catch unreasonable values. Locally on my machine I have seen
	// 30MB.
	assert.Equal(t, true, (self.RssKb*1024) < 5*TEN_MB)

	// If this is too little, feel free to increase it. But it should be low
	// enough to catch unreasonable values.
	assert.Equal(t, true, time.Since(self.startTime) < 30*time.Second)

	assert.Equal(t, false, self.CpuTime == nil)
	assert.Equal(t, true, *self.CpuTime >= 0)
	assert.Equal(t, true, *self.CpuTime < 30*time.Second)

	// Assume that only one test is running
	assert.Equal(t, self.DeduplicationSuffix, "")

	assert.SlicesEqual(t, self.children, []*Process{})

	assert.Equal(t, true, len(self.Cmdline) > 0)

	assert.Equal(t, true, *self.cpuPercent >= 0)

	assert.Equal(t, true, *self.memoryPercent >= 0)

	// Trace parents all the way up...
	proc := self
	for proc.parent != nil {
		proc = proc.parent
	}
	init := proc

	// ... and validate the init process.
	assert.Equal(t, 1, init.Pid)
	assert.Equal(t, 0, init.ppid)
	assert.Equal(t, true, len(init.children) > 0)
	assert.Equal(t, true, init.startTime.Before(self.startTime))
}

func TestRemoveSelfChildren_HidesHelpersButKeepsInteractiveChildren(t *testing.T) {
	self := &Process{Pid: 100, Cmdline: "ftop"}
	pager := &Process{Pid: 101, Cmdline: "less", parent: self}
	editor := &Process{Pid: 102, Cmdline: "Code", parent: pager}
	helper := &Process{Pid: 103, Cmdline: "ps", parent: self}

	self.children = []*Process{pager, helper}
	pager.children = []*Process{editor}

	processes := map[int]*Process{
		self.Pid:   self,
		pager.Pid:  pager,
		editor.Pid: editor,
		helper.Pid: helper,
	}

	removeSelfChildren(processes, self.Pid)

	_, helperFound := processes[helper.Pid]
	assert.Equal(t, false, helperFound)

	_, pagerFound := processes[pager.Pid]
	assert.Equal(t, true, pagerFound)

	_, editorFound := processes[editor.Pid]
	assert.Equal(t, true, editorFound)

	assert.SlicesEqual(t, self.children, []*Process{pager})
	assert.Equal(t, pager.parent, self)
	assert.Equal(t, editor.parent, pager)
}

func TestShouldHideSelfChild(t *testing.T) {
	assert.Equal(t, true, shouldHideSelfChild(&Process{Cmdline: "ps"}))
	assert.Equal(t, true, shouldHideSelfChild(&Process{Cmdline: "netstat"}))
	assert.Equal(t, true, shouldHideSelfChild(&Process{Cmdline: "iostat"}))

	assert.Equal(t, false, shouldHideSelfChild(&Process{Cmdline: "nano"}))
	assert.Equal(t, false, shouldHideSelfChild(&Process{Cmdline: "Code"}))
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
	assert.Equal(t, proc.lowercaseCommand, "sleep")

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
	assert.Equal(t, proc.lowercaseCommand, "bash")

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

func TestPsLineToProcess_IgnoresMonotonicClockInSameAs(t *testing.T) {
	lineA := "    1     0  3196       00:21     0  0.1 00:00:00  0.0 bash"
	lineB := "    1     0  3196       00:21     0  0.1 00:00:00  0.0 bash"

	baseSnapshot := time.Now()
	snapshotA := baseSnapshot

	// Emulate a wall-clock jump where wall time changed by 100ms while the
	// monotonic clock advanced by 9s.
	snapshotB := withWallFromAndMonotonicFrom(
		baseSnapshot.Add(100*time.Millisecond),
		baseSnapshot.Add(9*time.Second),
	)

	procA, err := psLineToProcess(lineA, snapshotA)
	assert.Equal(t, err, nil)

	procB, err := psLineToProcess(lineB, snapshotB)
	assert.Equal(t, err, nil)

	wallDelta := procA.startTime.Round(0).Sub(procB.startTime.Round(0)).Abs()
	assert.Equal(t, wallDelta <= 100*time.Millisecond, true)
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
	assert.Equal(t, proc.startTime, snapshotTime)
}

func TestProcessSameAs_AcceptsOneSecondDifference(t *testing.T) {
	base := time.Date(2026, time.March, 15, 8, 51, 33, 0, time.Local)

	proc := &Process{Pid: 1234, startTime: base}
	withinTolerance := &Process{Pid: 1234, startTime: base.Add(SAME_PROCESS_STARTTIME_TOLERANCE)}
	tooFarAway := &Process{Pid: 1234, startTime: base.Add(SAME_PROCESS_STARTTIME_TOLERANCE + time.Millisecond)}
	otherPid := &Process{Pid: 1235, startTime: base.Add(1 * time.Second)}

	assert.Equal(t, proc.SameAs(withinTolerance), true)
	assert.Equal(t, proc.SameAs(tooFarAway), false)
	assert.Equal(t, proc.SameAs(otherPid), false)
}

func TestGetExecutableForPid_Self(t *testing.T) {
	executable, err := getExecutableForPid(os.Getpid())
	assert.Equal(t, err, nil)
	assert.Equal(t, strings.TrimSpace(executable) != "", true)
}

func TestProcessCommandLine_FallsBackToExecutable(t *testing.T) {
	pid := os.Getpid()
	executable, err := getExecutableForPid(pid)
	assert.Equal(t, err, nil)

	process := &Process{
		Pid:     pid,
		Cmdline: "/tmp/\x00 broken",
	}

	commandLine := process.CommandLine()
	assert.SlicesEqual(t, commandLine, []string{executable})
}
