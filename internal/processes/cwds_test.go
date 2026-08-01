package processes

import (
	"os"
	"os/exec"
	"testing"

	"github.com/walles/ftop/internal/assert"
)

func TestLsofCwdParser(t *testing.T) {
	parser := lsofCwdParser{cwds: map[int]string{}}

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

// Directory names can contain newlines. lsof's NUL terminated output format
// tells us where the record really ends so we can put the name back together.
func TestLsofCwdParser_newlineInDirectoryName(t *testing.T) {
	parser := lsofCwdParser{cwds: map[int]string{}}

	assert.Equal(t, parser.parseLine("p1\x00"), nil)
	assert.Equal(t, parser.parseLine("fcwd\x00n/two"), nil)
	assert.Equal(t, parser.parseLine("lines\x00"), nil)

	assert.Equal(t, len(parser.cwds), 1)
	assert.Equal(t, parser.cwds[1], "/two\nlines")
}

// Files that aren't working directories should be ignored. lsof shouldn't send
// us any, but it costs us nothing to be sure.
func TestLsofCwdParser_ignoresOtherFiles(t *testing.T) {
	parser := lsofCwdParser{cwds: map[int]string{}}

	assert.Equal(t, parser.parseLine("p1\x00"), nil)
	assert.Equal(t, parser.parseLine("ftxt\x00n/usr/bin/cat\x00"), nil)

	assert.Equal(t, len(parser.cwds), 0)
}

// The real lsof should be able to tell us where we are.
func TestGetCwdsByPid(t *testing.T) {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof not available: ", err)
	}

	cwds, err := GetCwdsByPid()
	if err != nil {
		t.Fatalf("listing working directories failed: %v", err)
	}

	ourCwd, found := cwds[os.Getpid()]
	assert.Equal(t, found, true)
	assert.Equal(t, len(ourCwd) > 0, true)
}
