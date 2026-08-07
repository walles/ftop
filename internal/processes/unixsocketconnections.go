package processes

// The unix domain socket connections of proc, aggregated and sorted for display.
//
// FIXME: Stub, so that the tests can be reviewed before this is written.
func UnixSocketConnections(proc *Process, allProcesses []*Process, unixSocketsByPid map[int][]UnixSocket) []Connection {
	_, _, _ = proc, allProcesses, unixSocketsByPid

	return nil
}
