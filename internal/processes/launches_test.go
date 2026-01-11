package processes

import (
	"testing"

	"github.com/walles/ptop/internal/assert"
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

	didPanic := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
			}
		}()

		incrementLaunchCount(root, &p1)
	}()

	assert.Equal(t, didPanic, true)
}

func TestFlatten(t *testing.T) {
	manyLaunches := &LaunchNode{
		Command:     "many-launches",
		LaunchCount: 9,
		Children:    []*LaunchNode{},
	}
	fewLaunches := &LaunchNode{
		Command:     "few-launches",
		LaunchCount: 1,
		Children:    []*LaunchNode{},
	}

	// Test descending order by launch count, with inputs sorted
	flattened1 := (&LaunchNode{
		Command:  "root",
		Children: []*LaunchNode{manyLaunches, fewLaunches},
	}).Flatten()
	assert.Equal(t, len(flattened1), 2)
	assert.Equal(t, flattened1[0][1], manyLaunches)
	assert.Equal(t, flattened1[1][1], fewLaunches)

	// Test descending order by launch count, with inputs unsorted
	flattened2 := (&LaunchNode{
		Command:  "root",
		Children: []*LaunchNode{fewLaunches, manyLaunches},
	}).Flatten()
	assert.Equal(t, len(flattened2), 2)
	assert.Equal(t, flattened2[0][1], manyLaunches)
	assert.Equal(t, flattened2[1][1], fewLaunches)
}
