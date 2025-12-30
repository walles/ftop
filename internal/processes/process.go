package processes

import (
	"fmt"
	"time"
)

type Process struct {
	pid      int
	ppid     *int
	children []*Process
	parent   *Process

	cmdline           string
	command           string
	lowercase_command string

	start_time_string string
	age               time.Duration
	age_s             string

	username string

	rss_kb           int
	memory_percent   *float64
	memory_percent_s string

	cpu_percent           *float64
	cpu_time              *time.Duration
	cpu_time_s            string
	aggregated_cpu_time   time.Duration
	aggregated_cpu_time_s string
}

func (p *Process) String() string {
	return fmt.Sprintf("%s(%d)", p.command, p.pid)
}

func GetAll() ([]*Process, error) {
	FIXME
}
