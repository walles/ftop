package util

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// The command started, ran to completion and exited non-zero.
//
// Every line it did print was handed to the callback before this was returned,
// so a caller with a use for a partial result can carry on with what it got.
// Some commands, lsof above all, exit non-zero over things they still report
// around.
//
// Exec() returns this type for that case alone, and a plain error for a command
// it couldn't start or whose output wouldn't parse. Those leave nothing to carry
// on with, so errors.As() on this type is how a caller tells the two apart.
type ExitError struct {
	commandName string
	err         error
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("%s command failed: %v", e.commandName, e.err)
}

func (e *ExitError) Unwrap() error {
	return e.err
}

// The error for a command that ran and then failed, err being what cmd.Wait()
// returned.
//
// An ExitError for a command that exited under its own steam, a plain error for
// one a signal took down.
func waitError(commandName string, err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.Exited() {
		return &ExitError{commandName: commandName, err: err}
	}

	return fmt.Errorf("%s command failed: %v", commandName, err)
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
			readErr = waitError(commandline[0], err)
		}
	}

	if readErr != nil {
		return readErr
	}

	return nil
}
