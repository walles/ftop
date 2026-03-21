package main

import (
	"testing"

	"github.com/walles/ftop/internal/assert"
)

func resetCLI() {
	CLI = commandLine{}
}

func TestParseCommandLine_InitialFilter(t *testing.T) {
	resetCLI()
	t.Cleanup(resetCLI)

	argsParser, err := newArgsParser()
	assert.Equal(t, err, nil)

	_, err = argsParser.Parse([]string{"firefox"})
	assert.Equal(t, err, nil)

	assert.Equal(t, CLI.InitialFilter, "firefox")
	assert.Equal(t, CLI.Theme.String(), "auto")
}

func TestParseCommandLine_WithoutInitialFilter(t *testing.T) {
	resetCLI()
	t.Cleanup(resetCLI)

	argsParser, err := newArgsParser()
	assert.Equal(t, err, nil)

	_, err = argsParser.Parse([]string{})
	assert.Equal(t, err, nil)

	assert.Equal(t, CLI.InitialFilter, "")
	assert.Equal(t, CLI.Theme.String(), "auto")
}
