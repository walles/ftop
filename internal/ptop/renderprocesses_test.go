package ptop

import (
	"reflect"
	"testing"
	"time"

	"github.com/walles/ptop/internal/assert"
	"github.com/walles/ptop/internal/processes"
)

func TestCreateProcessTable(t *testing.T) {
	sortedProcs := []processes.Process{
		{Pid: 6, Command: "six", Username: "six", RssKb: 60, CpuTime: toDuration(60)},
		{Pid: 5, Command: "five", Username: "five", RssKb: 50, CpuTime: toDuration(50)},
		{Pid: 4, Command: "four", Username: "four", RssKb: 40, CpuTime: toDuration(40)},
		{Pid: 3, Command: "three", Username: "three", RssKb: 30, CpuTime: toDuration(30)},
		{Pid: 2, Command: "two", Username: "two", RssKb: 20, CpuTime: toDuration(20)},
		{Pid: 1, Command: "one", Username: "one", RssKb: 10, CpuTime: toDuration(10)},
	}

	table, usersHeight, returnedSortedProcs, users, binaries := createProcessesTable(sortedProcs, 6)

	assert.Equal(t, usersHeight, 2) // Header line + 1 user line
	assert.Equal(t, reflect.DeepEqual(returnedSortedProcs, sortedProcs), true)
	assert.SlicesEqual(t, binaries, []binaryStats{
		{stats{name: "six", cpuTime: 60000000000, rssKb: 60, processCount: 1}},
		{stats{name: "five", cpuTime: 50000000000, rssKb: 50, processCount: 1}},
		{stats{name: "four", cpuTime: 40000000000, rssKb: 40, processCount: 1}},
		{stats{name: "three", cpuTime: 30000000000, rssKb: 30, processCount: 1}},
		{stats{name: "two", cpuTime: 20000000000, rssKb: 20, processCount: 1}},
		{stats{name: "one", cpuTime: 10000000000, rssKb: 10, processCount: 1}},
	})
	assert.SlicesEqual(t, users, []userStats{
		{stats{name: "six", cpuTime: 60000000000, rssKb: 60, processCount: 1}},
		{stats{name: "five", cpuTime: 50000000000, rssKb: 50, processCount: 1}},
		{stats{name: "four", cpuTime: 40000000000, rssKb: 40, processCount: 1}},
		{stats{name: "three", cpuTime: 30000000000, rssKb: 30, processCount: 1}},
		{stats{name: "two", cpuTime: 20000000000, rssKb: 20, processCount: 1}},
		{stats{name: "one", cpuTime: 10000000000, rssKb: 10, processCount: 1}},
	})

	assert.Equal(t, reflect.DeepEqual(table, [][]string{
		{"PID", "Command", "Username", "CPU", "Time", "RAM", "Username", "CPU", "RAM"},
		{"6", "six", "six", "--", "1m00s", "60k", "six", "1m00s", "60k"},
		{"5", "five", "five", "--", "50.00s", "50k", "", "", ""},
		{"4", "four", "four", "--", "40.00s", "40k", "", "", ""},
		{"3", "three", "three", "--", "30.00s", "30k", "Binary", "CPU", "RAM"},
		{"2", "two", "two", "--", "20.00s", "20k", "six", "1m00s", "60k"},
	}), true)
}

func toDuration(seconds int) *time.Duration {
	d := time.Duration(seconds) * time.Second
	return &d
}
