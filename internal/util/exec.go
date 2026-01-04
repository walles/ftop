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
	cmd := exec.Command(commandline[0], commandline[1:]...)

	env := []string{}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "LANG") || strings.HasPrefix(e, "LC_") {
			continue
		}
		env = append(env, e)
	}
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Failed to get stdout pipe for %s: %v", commandline[0], err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Failed to start %s: %v", commandline[0], err)
	}

	scanner := bufio.NewScanner(stdout)
	var readErr error
	for scanner.Scan() {
		line := scanner.Text()

		err := perLineCallback(line)
		if err != nil {
			if readErr == nil {
				readErr = fmt.Errorf("Failed to parse %s line: %v", commandline[0], err)
			}
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		if readErr == nil {
			readErr = fmt.Errorf("Error reading %s output: %v", commandline[0], err)
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
