package processes

// The pipes proc holds an end of, aggregated and sorted for display.
//
// pipeEndsByPid is every pipe end we could see, see GetPipeEndsByPid(); the ends
// held by other processes are what lets us name the process at the other end of
// a pipe. allProcesses turns those PIDs into names; pass every process you know
// about. A peer we find no process for keeps its PID and gets no name.
//
// Pipe ends we find no peer for are left out. The peer may be a process we
// aren't allowed to inspect, or the pipe may have nobody at the other end at
// all, and there is no telling those apart.
//
// Every connection comes back with Protocol ProtocolPipe and Port 0, a pipe
// having no port.
func PipeConnections(proc *Process, allProcesses []*Process, pipeEndsByPid map[int][]PipeEnd) []Connection {
	// TODO: Implement
	return nil
}
