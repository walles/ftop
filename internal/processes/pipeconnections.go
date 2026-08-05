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
//
// Unlike a socket, a pipe has as many ends as anybody cares to fork, so every
// peer holding one gets a line of its own: three processes reading what we write
// are three processes we are talking to. A process holding both ends of a pipe
// is the one exception, that being one pipe and so one line, see
// isTheReportingEnd().
//
// Count is how many distinct pipes a line stands for, so two processes at the
// ends of one pipe report a count of 1 however many file descriptors either of
// them has it open on.
//
// Known limit: two ends of one pipe held by us can disagree about its direction,
// an end open for reading and writing both being DirectionUnknown where a plain
// write end of the same pipe is DirectionOutgoing. Direction is part of what
// identifies a line, so such a pipe draws two lines to the one peer rather than
// one. It takes "exec 3>fifo 4<>fifo" to construct.
func PipeConnections(proc *Process, allProcesses []*Process, pipeEndsByPid map[int][]PipeEnd) []Connection {
	if len(pipeEndsByPid[proc.Pid]) == 0 {
		return nil
	}

	// Deduplicated up front rather than inside the loop below, which would walk
	// the whole machine's pipe ends again for every end we hold ourselves. That
	// is thousands of ends times dozens of our own on the busy box this is for.
	endsByPid := make(map[int][]PipeEnd, len(pipeEndsByPid))
	for pid, ends := range pipeEndsByPid {
		endsByPid[pid] = deduplicatePipeEnds(ends)
	}

	ourEnds := endsByPid[proc.Pid]

	names := map[int]string{}
	for _, candidate := range allProcesses {
		names[candidate.Pid] = candidate.Command()
	}

	// The connections we found, mapped to the pipes carrying each of them. A set
	// rather than a counter because one pipe can match several of a peer's ends,
	// a process holding a FIFO for reading and for writing both being enough,
	// and that is still one pipe. The keys carry no Count of their own, that is
	// what the values are for.
	pipes := map[Connection]map[string]bool{}

	for _, ourEnd := range ourEnds {
		for peerPid, theirEnds := range endsByPid {
			for _, theirEnd := range theirEnds {
				if !arePipeEnds(ourEnd, theirEnd) {
					continue
				}

				if peerPid == proc.Pid && !isTheReportingEnd(ourEnd, theirEnd) {
					continue
				}

				connection := Connection{
					Peer:      Peer{Name: names[peerPid], Pid: peerPid},
					Protocol:  ProtocolPipe,
					Direction: pipeDirection(ourEnd),
				}

				if pipes[connection] == nil {
					pipes[connection] = map[string]bool{}
				}

				pipes[connection][pipeIdentity(ourEnd)] = true
			}
		}
	}

	connections := make([]Connection, 0, len(pipes))
	for connection, carriedBy := range pipes {
		connection.Count = len(carriedBy)
		connections = append(connections, connection)
	}

	SortConnections(connections)

	return connections
}

// True if ours and theirs are two ends of one and the same pipe, so that what is
// written into one can be read out of the other.
//
// Two mechanisms in one predicate, needing no GOOS switch because only one of
// them can fire for any given end. The device clause tests PeerDevice, which is
// populated from an "n->0x..." name and nothing else, and only macOS names an
// anonymous pipe end that way; Linux calls every anonymous pipe the literal
// string "pipe" and identifies it by its inode instead. A named FIFO carries an
// inode on both platforms and goes the inode way.
//
// PeerDevice rather than Device is what carries that argument. A FIFO can have a
// device of its own — the file system it lives on, which is how Linux reports
// one — so Device being set says nothing about which clause applies.
//
// The inode way needs the access modes as well, since every end of a pipe shares
// its inode and two processes writing into one pipe are not talking to each
// other. The device way needs no such thing, an end named that way pointing at
// exactly one other end.
//
// This is no check that the two ends are distinct: an end open for reading and
// writing both satisfies it against itself. A caller walking one process' own
// ends has to exclude that, which PipeConnections() does via isTheReportingEnd().
//
// Known limit: two named FIFOs on different file systems can share an inode
// number, and this would call their ends a pair. Telling them apart means
// comparing the device as well, which macOS reports for no FIFO at all.
func arePipeEnds(ours PipeEnd, theirs PipeEnd) bool {
	if ours.PeerDevice != "" && ours.PeerDevice == theirs.Device {
		return true
	}

	if ours.Inode == "" || ours.Inode != theirs.Inode {
		return false
	}

	// An end that can write pairs with an end that can read, which makes an end
	// open for both a peer of readers, writers and other such ends alike. An end
	// lsof reports no mode for can do neither as far as we know, and pairs with
	// nothing.
	return canWrite(ours) && canRead(theirs) || canRead(ours) && canWrite(theirs)
}

// Which way the data goes through the end we hold: out of us when we write into
// it, into us when we read from it.
//
// DirectionUnknown for an end that can go either way, and for one lsof names no
// mode for, which is every anonymous pipe on macOS.
func pipeDirection(end PipeEnd) Direction {
	switch end.Access {
	case PipeAccessWrite:
		return DirectionOutgoing

	case PipeAccessRead:
		return DirectionIncoming

	default:
		return DirectionUnknown
	}
}

// True if ours is the end to report a pipe by, where both of its ends are held
// by the process we are reporting on.
//
// Such a pipe is one pipe and deserves one line, so one of the two ends has to
// stand for it. The writing one does, data flowing from there, and where lsof
// won't say which end writes it goes by whichever end sorts first: arbitrary,
// but the same end every time the page is opened.
//
// False for an end against itself, which is also what keeps a lone end open for
// reading and writing both out of the listing as its own peer: arePipeEnds()
// says such an end pairs with itself, and this is where that gets dropped.
func isTheReportingEnd(ours PipeEnd, theirs PipeEnd) bool {
	if canWrite(ours) != canWrite(theirs) {
		return canWrite(ours)
	}

	return pipeEndIdentity(ours) < pipeEndIdentity(theirs)
}

func canWrite(end PipeEnd) bool {
	return end.Access == PipeAccessWrite || end.Access == PipeAccessReadWrite
}

func canRead(end PipeEnd) bool {
	return end.Access == PipeAccessRead || end.Access == PipeAccessReadWrite
}

// What identifies the pipe an end belongs to, for telling several pipes to one
// peer apart from one pipe that matched several of its ends.
//
// The inode where there is one, that being the pipe itself. Where there isn't,
// which is macOS for an anonymous pipe, the kernel addresses of the pipe's two
// ends name it just as well, sorted so that either end spells it the same way.
//
// Known limit, the same one arePipeEnds() has: two named FIFOs on different file
// systems sharing an inode number come out as one pipe here, so a peer at the
// end of both gets a Count of 1 rather than 2. Closing it means keying on the
// device as well.
func pipeIdentity(end PipeEnd) string {
	if end.Inode != "" {
		return "inode " + end.Inode
	}

	return "devices " + min(end.Device, end.PeerDevice) + " " + max(end.Device, end.PeerDevice)
}

// A process' pipe ends with the repeats left out, recognized by what they are
// rather than by the descriptor they arrived on.
//
// One end open on several file descriptors, by dup(2) or by being inherited as
// both stdout and stderr, is reported once per descriptor, and counting those
// would report one pipe as several. What is left is how lsof names the end plus
// how it is open, which is everything about it that isn't the descriptor.
func deduplicatePipeEnds(ends []PipeEnd) []PipeEnd {
	seen := map[string]bool{}

	var deduplicated []PipeEnd
	for _, end := range ends {
		identity := pipeEndIdentity(end)
		if seen[identity] {
			continue
		}

		seen[identity] = true
		deduplicated = append(deduplicated, end)
	}

	return deduplicated
}

// Everything about a pipe end except which file descriptor it arrived on, in a
// form that can be compared and ordered.
func pipeEndIdentity(end PipeEnd) string {
	return string(end.Access) + "\x00" + end.Device + "\x00" + end.PeerDevice + "\x00" + end.Inode
}
