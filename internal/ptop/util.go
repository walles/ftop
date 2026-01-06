package ptop

import (
	"os/user"

	"github.com/walles/ptop/internal/log"
)

// Or the empty string if lookup fails
func getCurrentUserName() string {
	currentUser, err := user.Current()
	if err != nil {
		log.Infof("Failed to look up current user: %v", err)
		return ""
	}

	return currentUser.Username
}
