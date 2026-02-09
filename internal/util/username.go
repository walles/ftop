package util

import (
	"os/user"

	"github.com/walles/ftop/internal/log"
)

// Or the empty string if lookup fails
func GetCurrentUsername() string {
	currentUser, err := user.Current()
	if err != nil {
		log.Infof("Failed to look up current user: %v", err)
		return ""
	}

	return currentUser.Username
}
