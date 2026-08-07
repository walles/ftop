package util

import (
	"errors"
	"testing"

	"github.com/walles/ftop/internal/assert"
)

func ignoreLine(_ string) error {
	return nil
}

// A command that ran and chose to exit non-zero gets an ExitError, which is what
// lets a caller keep the output that arrived before the exit.
func TestExec_nonZeroExit(t *testing.T) {
	err := Exec([]string{"false"}, ignoreLine)

	var exitError *ExitError
	assert.Equal(t, errors.As(err, &exitError), true)
}

// A command that never started printed nothing, so there is nothing for a caller
// to carry on with and the error says so rather than being an ExitError.
func TestExec_startFailure(t *testing.T) {
	err := Exec([]string{"ftop-no-such-command"}, ignoreLine)

	var exitError *ExitError
	assert.Equal(t, err != nil, true)
	assert.Equal(t, errors.As(err, &exitError), false)
}

// A command a signal took down never reached the end of its output, so what it
// did print is as suspect as its death. That makes it a not-ExitError.
//
// The point with this distinction is that lsof tends to spontaneously exit with
// 1 whenever nothing is found, even if there was no problem. We want to be able
// to tell "exit 1, use the output" from "killed by signal, don't use the
// output".
func TestExec_killedBySignal(t *testing.T) {
	err := Exec([]string{"sh", "-c", "kill -TERM $$"}, ignoreLine)

	var exitError *ExitError
	assert.Equal(t, err != nil, true)
	assert.Equal(t, errors.As(err, &exitError), false)
}
