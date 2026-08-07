package util

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// True for an error from Exec() meaning the command ran and exited non-zero
// under its own steam. False for one a signal took down, and for one that never
// started.
//
// Every line the command did print was handed to the callback before that error
// came back, so this is what tells a caller its partial result is worth keeping.
// Some commands, lsof above all, exit non-zero over things they still report
// around.
func IsExitStatus(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.Exited()
}

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
			// Wrapped, so that IsExitStatus() can see what kind of failure it was
			readErr = fmt.Errorf("%s command failed: %w", commandline[0], err)
		}
	}

	if readErr != nil {
		return readErr
	}

	return nil
}
