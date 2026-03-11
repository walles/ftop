package ftop

import (
	"strings"

	"github.com/walles/ftop/internal/processes"
	"github.com/walles/moor/v2/pkg/moor"
)

func pageProcessInfo(proc *processes.Process) error {
	if proc == nil {
		panic("proc is nil, can't page process info")
	}

	lines := strings.Join(proc.CommandLine(), "\n  ")
	return moor.PageFromString(lines, moor.Options{NoLineNumbers: true})
}
