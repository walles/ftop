package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Exec command line using the default locale and invokes the callback for each
// line.
func Exec(commandline []string, perLineCallback func(line string) error) error {
	env := []string{}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "LANG") || strings.HasPrefix(e, "LC_") {
			continue
		}
		env = append(env, e)
	}

	return execWithEnv(commandline, env, perLineCallback)
}

// Like Exec(), but with the user's own locale left in place.
//
// Use this for commands printing file names. Without a locale telling them
// which character encoding to use, some commands escape every non-ASCII byte
// into something unreadable.
func ExecInUsersLocale(commandline []string, perLineCallback func(line string) error) error {
	return execWithEnv(commandline, os.Environ(), perLineCallback)
}

func execWithEnv(commandline []string, env []string, perLineCallback func(line string) error) error {
	cmd := exec.Command(commandline[0], commandline[1:]...)
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe for %s: %v", commandline[0], err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %v", commandline[0], err)
	}

	scanner := bufio.NewScanner(stdout)
	var readErr error
	for scanner.Scan() {
		line := scanner.Text()

		err := perLineCallback(line)
		if err != nil {
			if readErr == nil {
				readErr = fmt.Errorf("failed to parse %s line: %v", commandline[0], err)
			}
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		if readErr == nil {
			readErr = fmt.Errorf("error reading %s output: %v", commandline[0], err)
		}
	}

	if err := cmd.Wait(); err != nil {
		if readErr == nil {
			readErr = fmt.Errorf("%s command failed: %v", commandline[0], err)
		}
	}

	if readErr != nil {
		return readErr
	}

	return nil
}
