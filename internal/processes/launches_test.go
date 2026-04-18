package processes

import (
	"testing"
	"time"

	"github.com/walles/ftop/internal/assert"
)

func TestIncrementLaunchCount_fromScratch(t *testing.T) {
	newRoot := incrementLaunchCount(nil, &Process{
		Pid:     1,
		Cmdline: "init",
	})
	assertAncestry(t, newRoot, ancestry{Command: "init", LaunchCount: 1})

	newRoot = incrementLaunchCount(newRoot, &Process{
		Pid:     1,
		Cmdline: "init",
	})
	assertAncestry(t, newRoot, ancestry{Command: "init", LaunchCount: 2})
}

func TestIncrementLaunchCount_oneDown(t *testing.T) {
	p0 := Process{Pid: 1, Cmdline: "init"}
	p1 := Process{Cmdline: "sshd", parent: &p0}

	root := &LaunchNode{Command: "init"}
	newRoot := incrementLaunchCount(root, &p1)
	assert.Equal(t, root, newRoot) // Same root object
	assertAncestry(t, root, ancestry{
		Command:     "init",
		LaunchCount: 0,
		Children: []ancestry{
			{Command: "sshd", LaunchCount: 1},
		},
	})
}

func TestIncrementLaunchCount_twoChildren(t *testing.T) {
	p0 := Process{Pid: 1, Cmdline: "init"}
	p1 := Process{Cmdline: "sshd", parent: &p0}
	p2 := Process{Cmdline: "telnetd", parent: &p0}

	root := &LaunchNode{Command: "init"}
	newRoot := incrementLaunchCount(root, &p1)
	assert.Equal(t, root, newRoot) // Same root object
	newRoot = incrementLaunchCount(root, &p2)
	assert.Equal(t, root, newRoot) // Still same root object
	assertAncestry(t, root, ancestry{
		Command:     "init",
		LaunchCount: 0,
		Children: []ancestry{
			{Command: "sshd", LaunchCount: 1},
			{Command: "telnetd", LaunchCount: 1},
		},
	})
}

func TestIncrementLaunchCount_sameChildTwice(t *testing.T) {
	p0 := Process{Pid: 1, Cmdline: "init"}
	p1 := Process{Cmdline: "sshd", parent: &p0}

	root := &LaunchNode{Command: "init"}
	newRoot := incrementLaunchCount(root, &p1)
	assert.Equal(t, root, newRoot) // Same root object
	newRoot = incrementLaunchCount(root, &p1)
	assert.Equal(t, root, newRoot) // Still same root object
	assertAncestry(t, root, ancestry{
		Command:     "init",
		LaunchCount: 0,
		Children: []ancestry{
			{Command: "sshd", LaunchCount: 2},
		},
	})
}

func TestIncrementLaunchCount_twoStepsToLeaf(t *testing.T) {
	p0 := Process{Pid: 1, Cmdline: "init"}
	p1 := Process{Cmdline: "sshd", parent: &p0}
	p2 := Process{Cmdline: "bash", parent: &p1}

	root := &LaunchNode{Command: "init"}
	newRoot := incrementLaunchCount(root, &p2)
	assert.Equal(t, root, newRoot) // Same root object
	assertAncestry(t, root, ancestry{
		Command:     "init",
		LaunchCount: 0,
		Children: []ancestry{
			{
				Command:     "sshd",
				LaunchCount: 0,
				Children: []ancestry{
					{Command: "bash", LaunchCount: 1},
				},
			},
		},
	})
}

func TestIncrementLaunchCount_noState(t *testing.T) {
	root0 := incrementLaunchCount(nil, &Process{
		Pid:     1,
		Cmdline: "init",
	})
	root1 := incrementLaunchCount(nil, &Process{
		Cmdline: "sshd",
		parent:  &Process{Pid: 1, Cmdline: "init"},
	})

	// Prevent state leakage between calls
	assert.Equal(t, root0 == root1, false)
}

// "init" -> "init" should create a child under the root "init". The top one
// should have launch count 0, the bottom one 1.
func TestIncrementLaunchCount_initWithAnotherInitChild(t *testing.T) {
	p0 := Process{Pid: 1, Cmdline: "init"}
	p1 := Process{Cmdline: "init", parent: &p0}

	root := incrementLaunchCount(nil, &p1)
	assertAncestry(t, root, ancestry{
		Command:     "init",
		LaunchCount: 0,
		Children: []ancestry{
			{Command: "init", LaunchCount: 1},
		},
	})

	root = incrementLaunchCount(root, &p1)
	assertAncestry(t, root, ancestry{
		Command:     "init",
		LaunchCount: 0,
		Children: []ancestry{
			{Command: "init", LaunchCount: 2},
		},
	})
}

func TestIncrementLaunchCount_parentlessNonInitGoesUnderVirtualParent(t *testing.T) {
	p0 := Process{Cmdline: "sshd"}

	root := incrementLaunchCount(nil, &p0)
	assertAncestry(t, root, ancestry{
		Command:     "init",
		LaunchCount: 0,
		Children: []ancestry{
			{
				Command:     "...",
				LaunchCount: 0,
				Children: []ancestry{
					{Command: "sshd", LaunchCount: 1},
				},
			},
		},
	})
}

type ancestry struct {
	Pid         int
	Command     string
	LaunchCount int
	Children    []ancestry
}

func assertAncestry(t *testing.T, node *LaunchNode, expected ancestry) {
	t.Helper()

	if node == nil {
		t.Fatalf("expected node <%s> but got <nil>", expected.Command)
	}

	if node.Command != expected.Command {
		t.Fatalf("expected command <%s> but got <%s>", expected.Command, node.Command)
	}

	if node.LaunchCount != expected.LaunchCount {
		t.Fatalf("expected launch count %d for <%s> but got %d", expected.LaunchCount, expected.Command, node.LaunchCount)
	}

	if len(node.Children) != len(expected.Children) {
		t.Fatalf("expected %d children under <%s> but got %d", len(expected.Children), expected.Command, len(node.Children))
	}

	for i, expectedChild := range expected.Children {
		assertAncestry(t, node.Children[i], expectedChild)
	}
}

func TestIncrementLaunchCount_mismatchedRootDoesNotPanic(t *testing.T) {
	init := Process{Pid: 1, Cmdline: "launchd"}
	p1 := Process{Cmdline: "sshd", parent: &init}

	// Create a bad root different from ancestry[0]
	badRoot := &LaunchNode{Command: "bad root"}

	newRoot := incrementLaunchCount(badRoot, &p1)
	assert.Equal(t, newRoot.Command, "launchd")
	assert.Equal(t, newRoot.LaunchCount, 0)

	assertAncestry(t, newRoot, ancestry{
		Command:     "launchd",
		LaunchCount: 0,
		Children: []ancestry{
			{Command: "...", LaunchCount: 0, Children: []ancestry{{Command: "bad root"}}},
			{Command: "sshd", LaunchCount: 1},
		},
	})
}

func TestIncrementLaunchCount_pid1CommandIsDisplayed(t *testing.T) {
	p0 := Process{Pid: 1, Cmdline: "launchd"}
	p1 := Process{Cmdline: "sshd", parent: &p0}

	root := incrementLaunchCount(nil, &p1)
	assertAncestry(t, root, ancestry{
		Command:     "launchd",
		LaunchCount: 0,
		Children: []ancestry{
			{Command: "sshd", LaunchCount: 1},
		},
	})
}

// Reproduces the crash pattern seen in production where launch counting first
// sees a chain rooted in "code" and then a later launched process rooted in
// "launchd".
func TestUpdateLaunches_mixedRootsDoesNotPanic(t *testing.T) {
	previous := map[int]*Process{}
	current := map[int]*Process{}

	codeParent := &Process{Pid: 10331, Cmdline: "code", startTime: time.Now()}
	codeChild := &Process{Pid: 10340, Cmdline: "Code", parent: codeParent, startTime: time.Now()}
	current[codeChild.Pid] = codeChild

	root := updateLaunches(nil, buildProcessMatches(previous, current))
	assertAncestry(t, root, ancestry{
		Command:     "init",
		LaunchCount: 0,
		Children: []ancestry{
			{
				Command:     "...",
				LaunchCount: 0,
				Children: []ancestry{
					{
						Command:     "code",
						LaunchCount: 0,
						Children: []ancestry{
							{Command: "Code", LaunchCount: 1},
						},
					},
				},
			},
		},
	})

	previous = current
	current = map[int]*Process{}

	launchd := &Process{Pid: 1, Cmdline: "launchd", startTime: time.Now()}
	launchedEditor := &Process{Pid: 10720, Cmdline: "Code", parent: launchd, startTime: time.Now()}
	current[launchedEditor.Pid] = launchedEditor

	// This call used to panic
	root = updateLaunches(root, buildProcessMatches(previous, current))
	assertAncestry(t, root, ancestry{
		Command:     "launchd",
		LaunchCount: 0,
		Children: []ancestry{
			{
				Command:     "...",
				LaunchCount: 0,
				Children: []ancestry{
					{
						Command:     "code",
						LaunchCount: 0,
						Children: []ancestry{
							{Command: "Code", LaunchCount: 1},
						},
					},
				},
			},
			{Command: "Code", LaunchCount: 1},
		},
	})
}
