package processes

// Keeps track of all known processes, alive or now dead, and their names.
//
// If multiple process share the same name, the deduplicator can suggest a
// suffix to disambiguate them.
type deduplicator struct {
}

// Register a process as known. If it was alread known, the registration is a
// no-op.
//
// Processes are considered the same if they have the same PID and the same
// start time.
func (d *deduplicator) register(proc *Process) {

}

// Suggest a disambiguating string for this process. Example return values could be:
// - "" (the empty string) if there is only one of these
// - "1" if there are multiple ones with this name, and this is the oldest one
func (d *deduplicator) suffix(proc *Process) string {

}
