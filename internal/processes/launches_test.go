package processes

import (
	"testing"
	"time"

	"github.com/walles/ftop/internal/assert"
)

func TestIncrementLaunchCount_fromScratch(t *testing.T) {
	newRoot := incrementLaunchCount(nil, &Process{
		Command: "init",
	})

	assert.Equal(t, newRoot.LaunchCount, 1)
	assert.Equal(t, newRoot.Command, "init")
	assert.Equal(t, len(newRoot.Children), 0)

	newRoot = incrementLaunchCount(newRoot, &Process{
		Command: "init",
	})

	assert.Equal(t, newRoot.LaunchCount, 2)
	assert.Equal(t, newRoot.Command, "init")
	assert.Equal(t, len(newRoot.Children), 0)
}

func TestIncrementLaunchCount_oneDown(t *testing.T) {
	p0 := Process{Command: "init"}
	p1 := Process{Command: "sshd", parent: &p0}

	root := &LaunchNode{Command: "init"}
	newRoot := incrementLaunchCount(root, &p1)
	assert.Equal(t, root, newRoot) // Same root object

	assert.Equal(t, root.Command, "init")
	assert.Equal(t, root.LaunchCount, 0)
	assert.Equal(t, len(root.Children), 1)

	child := root.Children[0]
	assert.Equal(t, child.Command, "sshd")
	assert.Equal(t, child.LaunchCount, 1)
	assert.Equal(t, len(child.Children), 0)
}

func TestIncrementLaunchCount_twoChildren(t *testing.T) {
	p0 := Process{Command: "init"}
	p1 := Process{Command: "sshd", parent: &p0}
	p2 := Process{Command: "telnetd", parent: &p0}

	root := &LaunchNode{Command: "init"}
	newRoot := incrementLaunchCount(root, &p1)
	assert.Equal(t, root, newRoot) // Same root object
	newRoot = incrementLaunchCount(root, &p2)
	assert.Equal(t, root, newRoot) // Still same root object

	assert.Equal(t, root.Command, "init")
	assert.Equal(t, root.LaunchCount, 0)
	assert.Equal(t, len(root.Children), 2)

	child0 := root.Children[0]
	assert.Equal(t, child0.Command, "sshd")
	assert.Equal(t, child0.LaunchCount, 1)
	assert.Equal(t, len(child0.Children), 0)

	child1 := root.Children[1]
	assert.Equal(t, child1.Command, "telnetd")
	assert.Equal(t, child1.LaunchCount, 1)
	assert.Equal(t, len(child1.Children), 0)
}

func TestIncrementLaunchCount_sameChildTwice(t *testing.T) {
	p0 := Process{Command: "init"}
	p1 := Process{Command: "sshd", parent: &p0}

	root := &LaunchNode{Command: "init"}
	newRoot := incrementLaunchCount(root, &p1)
	assert.Equal(t, root, newRoot) // Same root object
	newRoot = incrementLaunchCount(root, &p1)
	assert.Equal(t, root, newRoot) // Still same root object

	assert.Equal(t, root.Command, "init")
	assert.Equal(t, root.LaunchCount, 0)
	assert.Equal(t, len(root.Children), 1)

	child := root.Children[0]
	assert.Equal(t, child.Command, "sshd")
	assert.Equal(t, child.LaunchCount, 2)
	assert.Equal(t, len(child.Children), 0)
}

func TestIncrementLaunchCount_twoStepsToLeaf(t *testing.T) {
	p0 := Process{Command: "init"}
	p1 := Process{Command: "sshd", parent: &p0}
	p2 := Process{Command: "bash", parent: &p1}

	root := &LaunchNode{Command: "init"}
	newRoot := incrementLaunchCount(root, &p2)
	assert.Equal(t, root, newRoot) // Same root object

	assert.Equal(t, root.Command, "init")
	assert.Equal(t, root.LaunchCount, 0)
	assert.Equal(t, len(root.Children), 1)

	child0 := root.Children[0]
	assert.Equal(t, child0.Command, "sshd")
	assert.Equal(t, child0.LaunchCount, 0)
	assert.Equal(t, len(child0.Children), 1)

	child1 := child0.Children[0]
	assert.Equal(t, child1.Command, "bash")
	assert.Equal(t, child1.LaunchCount, 1)
	assert.Equal(t, len(child1.Children), 0)
}

func TestIncrementLaunchCount_noState(t *testing.T) {
	root0 := incrementLaunchCount(nil, &Process{
		Command: "init",
	})
	root1 := incrementLaunchCount(nil, &Process{
		Command: "sshd",
		parent:  &Process{Command: "init"},
	})

	// Prevent state leakage between calls
	assert.Equal(t, root0 == root1, false)
}

// "init" -> "init" should create a child under the root "init". The top one
// should have launch count 0, the bottom one 1.
func TestIncrementLaunchCount_initWithAnotherInitChild(t *testing.T) {
	p0 := Process{Command: "init"}
	p1 := Process{Command: "init", parent: &p0}

	root := incrementLaunchCount(nil, &p1)

	// This is the top init
	assert.Equal(t, root.Command, "init")
	assert.Equal(t, root.LaunchCount, 0)
	assert.Equal(t, len(root.Children), 1)

	// This is the child init
	child := root.Children[0]
	assert.Equal(t, child.Command, "init")
	assert.Equal(t, child.LaunchCount, 1)
	assert.Equal(t, len(child.Children), 0)

	root = incrementLaunchCount(root, &p1)

	// This is the top init
	assert.Equal(t, root.Command, "init")
	assert.Equal(t, root.LaunchCount, 0)
	assert.Equal(t, len(root.Children), 1)

	// This is the child init, should now have two launch counts
	child = root.Children[0]
	assert.Equal(t, child.Command, "init")
	assert.Equal(t, child.LaunchCount, 2)
	assert.Equal(t, len(child.Children), 0)
}

func TestIncrementLaunchCount_mismatchedRootPanics(t *testing.T) {
	p0 := Process{Command: "init"}
	p1 := Process{Command: "sshd", parent: &p0}

	// Create a root with a different command than ancestry[0]
	root := &LaunchNode{Command: "different"}

	incrementLaunchCount(root, &p1)
}

// Reproduces the crash pattern seen in production where launch counting first
// sees a chain rooted in "code" and then a later launched process rooted in
// "launchd".
func TestUpdateLaunches_mixedRootsPanics(t *testing.T) {
	previous := map[int]*Process{}
	current := map[int]*Process{}

	codeParent := &Process{Pid: 10331, Command: "code", startTime: time.Now()}
	codeChild := &Process{Pid: 10340, Command: "Code", parent: codeParent, startTime: time.Now()}
	current[codeChild.Pid] = codeChild

	root := updateLaunches(nil, previous, current)
	assert.Equal(t, root.Command, "code")

	previous = current
	current = map[int]*Process{}

	launchd := &Process{Pid: 1, Command: "launchd", startTime: time.Now()}
	launchedEditor := &Process{Pid: 10720, Command: "Code", parent: launchd, startTime: time.Now()}
	current[launchedEditor.Pid] = launchedEditor

	// We expect this line not to panic
	updateLaunches(root, previous, current)
}
