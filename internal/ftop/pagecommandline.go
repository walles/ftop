package ftop

import (
	"fmt"
	"os"
	"strings"

	"github.com/walles/ftop/internal/processes"
)

func (u *Ui) commandLineForPaging(proc *processes.Process, pt *pageText) {
	pt.writeTitle("Command Line for " + proc.String())

	split := proc.DisplayCommandLine()
	if len(split) == 0 {
		panic(fmt.Sprintf("process has no command line: %s", proc.String()))
	}

	binary := split[0]
	lastSlashIndex := strings.LastIndex(binary, string(os.PathSeparator))
	var firstLine string
	if lastSlashIndex == -1 {
		firstLine = u.highlight(binary)
	} else {
		path := binary[0 : lastSlashIndex+1] // "/bin/"
		command := binary[lastSlashIndex+1:] // "ls"
		firstLine = path + u.highlight(command)
	}

	pt.writeLine(firstLine)

	noMoreOptions := false
	lastWasOption := false
	for _, arg := range split[1:] {
		if arg == "--" {
			noMoreOptions = true
		}

		extraIndent := ""
		if lastWasOption {
			extraIndent = "  "
		}
		if strings.HasPrefix(arg, "-") {
			lastWasOption = true
			extraIndent = ""
		} else {
			lastWasOption = false
		}
		if noMoreOptions {
			extraIndent = ""
		}

		pt.writeLine("  " + extraIndent + arg)
	}
}
