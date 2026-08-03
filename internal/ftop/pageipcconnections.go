package ftop

import (
	"github.com/walles/ftop/internal/processes"
	"github.com/walles/moor/v2/twin"
)

func (u *Ui) ipcConnectionsForPaging(
	currentProcess *processes.Process,
	pt *pageText,
) {
	const title = "Inter Process Communication"

	pt.writeTitle(title)

	boldPrefix := twin.StyleDefault.WithAttr(twin.AttrBold).RenderUpdateFrom(twin.StyleDefault, twin.ColorCount24bit)
	boldSuffix := twin.StyleDefault.RenderUpdateFrom(twin.StyleDefault.WithAttr(twin.AttrBold), twin.ColorCount24bit)
	boldProc := boldPrefix + currentProcess.String() + boldSuffix

	// stdout piping into some other process
	pt.writeLine("                   " + boldProc + " | sort(1234)")

	// stdin coming from some other process
	pt.writeLine("      grep(1234) | " + boldProc)

	// both stdin and stdout connected to some other process
	pt.writeLine("      grep(1234) | " + boldProc + " | sort(1234)")

	// any other fd piping to some other process
	pt.writeLine("                   " + boldProc + " pipe to sort(1234)")

	// pipe into not-stdin
	pt.writeLine("grep(1234) pipe to " + boldProc)
}
