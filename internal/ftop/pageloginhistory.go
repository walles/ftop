package ftop

import (
	"github.com/walles/ftop/internal/loginhistory"
	"github.com/walles/ftop/internal/processes"
)

// Seam for testing, see the ones in pageprocessinfo.go
var getLoggedInUsersAt = loginhistory.GetUsersAt

func (u *Ui) usersLoggedInWhenProcessStartedForPaging(proc *processes.Process, pt *pageText) {
	pt.writeTitle("Users logged in when " + proc.String() + " started")

	users, err := getLoggedInUsersAt(proc.StartTime())
	if err != nil {
		pt.writeLine("<Unable to inspect login history: " + err.Error() + ">")
		return
	}

	if len(users) == 0 {
		pt.writeLine("<Nobody found, either nobody was logged in or the wtmp logs have been rotated>")
		return
	}

	for _, user := range users {
		pt.writeLine(user)
	}
}
