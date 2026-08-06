package ftop

import (
	"github.com/walles/ftop/internal/processes"
)

// Seam for testing, see the ones in pageprocessinfo.go
var getCwdsByPid = processes.GetCwdsByPid

func (u *Ui) cwdFriendsForPaging(
	currentProcess *processes.Process,
	allProcesses []*processes.Process,
	pt *pageText,
) {
	const title = "Others sharing this process' working directory"

	cwds, err := getCwdsByPid()
	if err != nil {
		pt.writeTitle(title)
		pt.writeLine("<Unable to list working directories: " + err.Error() + ">")
		return
	}

	cwd, found := cwds[currentProcess.Pid]
	if !found {
		pt.writeTitle(title)
		pt.writeLine("<Working directory unknown, try again or try \"sudo ftop\">")
		return
	}

	pt.writeTitle(title + " (" + cwd + ")")

	if cwd == "/" {
		pt.writeLine("<Working directory too common, never mind>")
		return
	}

	friends := processes.CwdFriends(currentProcess, allProcesses, cwds)
	if len(friends) == 0 {
		pt.writeLine("<Nobody else shares this working directory>")
		return
	}

	for _, friend := range friends {
		pt.writeLine(friend.String())
	}
}
