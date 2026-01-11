package processes

import (
	"cmp"
	"fmt"
	"slices"
)

// Tracks processes launched while ptop is running. Will be rendered as a tree,
// with counts per node. Each node in the tree has a command name, a launch
// count and a list of child nodes.
type LaunchNode struct {
	Command     string
	LaunchCount int
	Children    []*LaunchNode
}

func updateLaunches(root *LaunchNode, previous, current map[int]*Process) *LaunchNode {
	for _, proc := range current {
		if samePidBefore, existed := previous[proc.Pid]; existed {
			if samePidBefore.startTime.Equal(proc.startTime) {
				// Same PID, same start time, same process as before, nothing to do
				continue
			}
		}

		// This process was launched since last update
		root = incrementLaunchCount(root, proc)
	}

	return root
}

func incrementLaunchCount(root *LaunchNode, newlyLaunched *Process) *LaunchNode {
	// Compute a parent chain like "init -> sshd -> bash"
	ancestry := []string{}
	for process := newlyLaunched; process != nil; process = process.parent {
		ancestry = append([]string{process.Command}, ancestry...)
	}

	// Ensure we have a root. If root is nil, create it from ancestry[0].
	if root == nil {
		root = &LaunchNode{Command: ancestry[0]}
	}

	node := root
	if ancestry[0] != node.Command {
		panic(fmt.Errorf("two different init commands reported, had <%s> then got ancestry %#v", node.Command, ancestry))
	}

	// Skip the root, it just got special treatment above ^
	for _, cmd := range ancestry[1:] {
		// find child matching command
		idx := slices.IndexFunc(node.Children, func(c *LaunchNode) bool {
			return c.Command == cmd
		})

		var child *LaunchNode
		if idx >= 0 {
			child = node.Children[idx]
		} else {
			child = &LaunchNode{Command: cmd}
			node.Children = append(node.Children, child)
		}

		node = child
	}

	// Increment the launch count on the leaf node
	node.LaunchCount++

	return root
}

// Convert tree to a slice of slices, which each internal slice representing one
// possible path from the root to a leaf. The slices will be sorted, with the
// slice with the highest max launch count first.
func (ln *LaunchNode) Flatten() [][]*LaunchNode {
	if ln == nil {
		return nil
	}

	var result [][]*LaunchNode
	var helper func(node *LaunchNode, path []*LaunchNode)
	helper = func(node *LaunchNode, path []*LaunchNode) {
		path = append(path, node)

		if len(node.Children) == 0 {
			// Leaf node, add path to result
			copiedPath := make([]*LaunchNode, len(path))
			copy(copiedPath, path)
			result = append(result, copiedPath)
			return
		}

		for _, child := range node.Children {
			helper(child, path)
		}
	}

	helper(ln, []*LaunchNode{})

	// Sort the result slices by the highest max launch count in each path, descending
	slices.SortFunc(result, func(a, b []*LaunchNode) int {
		maxA := slices.MaxFunc(a, func(n1, n2 *LaunchNode) int {
			return -cmp.Compare(n1.LaunchCount, n2.LaunchCount)
		}).LaunchCount

		maxB := slices.MaxFunc(b, func(n1, n2 *LaunchNode) int {
			return -cmp.Compare(n1.LaunchCount, n2.LaunchCount)
		}).LaunchCount

		return -cmp.Compare(maxA, maxB)
	})

	return result
}
