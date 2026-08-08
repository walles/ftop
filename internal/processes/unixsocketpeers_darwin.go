package processes

// Nothing to fill in: macOS lsof reports a unix domain socket's peer and its
// path itself, which is why this platform needs no second source. See the Linux
// implementation for the one that does.
func fillInPeersAndPaths(map[int][]UnixSocket) error {
	return nil
}
