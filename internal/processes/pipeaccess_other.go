//go:build !darwin

package processes

// Nothing to fill in: everywhere but macOS, lsof reports a pipe end's access
// mode itself. See the macOS implementation for the platform that needs a second
// source, and PipeAccess for what an access mode is for.
func fillInAccessModes(map[int][]PipeEnd) {
}
