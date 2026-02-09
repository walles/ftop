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
