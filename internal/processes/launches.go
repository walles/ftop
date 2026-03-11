package processes

import (
	"slices"
)

const launchRootFallbackCommand = "init"
const launchMissingParentCommand = "..."

// The root of the process tree always has PID 1. Because UNIX.
const initProcessPid = 1

// Tracks processes launched while ftop is running. Will be rendered as a tree,
// with counts per node. Each node in the tree has a command name, a launch
// count and a list of child nodes.
type LaunchNode struct {
	Command     string
	LaunchCount int
	Children    []*LaunchNode
}

// Keep launch counts tree up to date
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
	ancestry := []*Process{}
	for process := newlyLaunched; process != nil; process = process.parent {
		ancestry = append([]*Process{process}, ancestry...)
	}
	if len(ancestry) == 0 {
		return root
	}

	commands := make([]string, 0, len(ancestry))
	for _, process := range ancestry {
		commands = append(commands, process.Command)
	}

	rootCommand := launchRootFallbackCommand
	if ancestry[0].Pid == initProcessPid {
		rootCommand = ancestry[0].Command
	} else {
		if root != nil {
			rootCommand = root.Command
		}

		prefix := []string{rootCommand, launchMissingParentCommand}
		commands = append(prefix, commands...)
	}

	if len(commands) > 0 {
		commands[0] = rootCommand
	}

	// Ensure we have a root. If root is nil, create it from ancestry[0].
	if root == nil {
		root = &LaunchNode{Command: commands[0]}
	}
	if root.Command != commands[0] {
		if root.Command == launchRootFallbackCommand {
			root.Command = commands[0]
		}

		if root.Command != commands[0] {
			oldRoot := root
			root = &LaunchNode{Command: commands[0]}
			root.Children = append(root.Children, &LaunchNode{
				Command:  launchMissingParentCommand,
				Children: []*LaunchNode{oldRoot},
			})
		}
	}

	node := root

	// Skip the root, it just got special treatment above ^
	for _, cmd := range commands[1:] {
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
