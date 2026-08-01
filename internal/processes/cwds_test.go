package processes

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/walles/ftop/internal/assert"
)

func TestLsofCwdParser(t *testing.T) {
	parser := newLsofCwdParser()

	lines := []string{
		"p523\x00",
		"fcwd\x00n/\x00",
		"p4542\x00",
		"fcwd\x00n/Users/johan/src/ftop\x00",
	}
	for _, line := range lines {
		assert.Equal(t, parser.parseLine(line), nil)
	}

	assert.Equal(t, len(parser.cwds), 2)
	assert.Equal(t, parser.cwds[523], "/")
	assert.Equal(t, parser.cwds[4542], "/Users/johan/src/ftop")
}

// lsof escapes newlines in directory names rather than passing them through,
// so a name with a newline in it arrives as one single line.
func TestLsofCwdParser_newlineInDirectoryName(t *testing.T) {
	parser := newLsofCwdParser()

	assert.Equal(t, parser.parseLine("p1\x00"), nil)
	assert.Equal(t, parser.parseLine(`fcwd`+"\x00"+`n/two\nlines`+"\x00"), nil)

	assert.Equal(t, len(parser.cwds), 1)
	assert.Equal(t, parser.cwds[1], `/two\nlines`)
}

// Files that aren't working directories should be ignored. lsof shouldn't send
// us any, but it costs us nothing to be sure.
func TestLsofCwdParser_ignoresOtherFiles(t *testing.T) {
	parser := newLsofCwdParser()

	assert.Equal(t, parser.parseLine("p1\x00"), nil)
	assert.Equal(t, parser.parseLine("ftxt\x00n/usr/bin/cat\x00"), nil)

	assert.Equal(t, len(parser.cwds), 0)
}

// lsof can report files it has no name for. Those tell us nothing, and must
// not be mistaken for a bunch of processes all sharing the same directory.
func TestLsofCwdParser_ignoresNamelessFiles(t *testing.T) {
	parser := newLsofCwdParser()

	assert.Equal(t, parser.parseLine("p1\x00"), nil)
	assert.Equal(t, parser.parseLine("fcwd\x00n\x00"), nil)

	assert.Equal(t, len(parser.cwds), 0)
}

// The real lsof should be able to tell us where we are.
func TestGetCwdsByPid(t *testing.T) {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof not available: ", err)
	}

	expected, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	expected, err = filepath.EvalSymlinks(expected)
	if err != nil {
		t.Fatal(err)
	}

	cwds, err := GetCwdsByPid()
	if err != nil {
		t.Fatalf("listing working directories failed: %v", err)
	}

	assert.Equal(t, cwds[os.Getpid()], expected)
}
