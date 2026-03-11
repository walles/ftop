package processes

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/util"
)

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

	assert.Equal(t, true, self.Command == "processes.test" || strings.Contains(self.Command, "debug"))
	assert.Equal(t, self.lowercaseCommand, strings.ToLower(self.Command))

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

	assert.Equal(t, true, len(self.cmdline) > 0)

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
	self := &Process{Pid: 100, Command: "ftop"}
	pager := &Process{Pid: 101, Command: "less", parent: self}
	editor := &Process{Pid: 102, Command: "Code", parent: pager}
	helper := &Process{Pid: 103, Command: "ps", parent: self}

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
	assert.Equal(t, true, shouldHideSelfChild(&Process{Command: "ps"}))
	assert.Equal(t, true, shouldHideSelfChild(&Process{Command: "netstat"}))
	assert.Equal(t, true, shouldHideSelfChild(&Process{Command: "iostat"}))

	assert.Equal(t, false, shouldHideSelfChild(&Process{Command: "nano"}))
	assert.Equal(t, false, shouldHideSelfChild(&Process{Command: "Code"}))
}
