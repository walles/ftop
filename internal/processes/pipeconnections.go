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
// them has it open on, and however many of its ends either of them holds.
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

	// One entry per pipe we share with a peer, holding what our own ends of that
	// pipe let us do with it. Keyed by peer and pipe rather than by the Connection
	// the end produces, so that several of our own ends of one pipe make one line:
	// they can disagree about direction, an end open for reading and writing both
	// saying nothing where a write end of that very pipe says outgoing, and the
	// pipe is one pipe regardless.
	//
	// One pipe can also match several of the peer's ends, a process holding a FIFO
	// for reading and for writing both being enough, and this collapses that as
	// well.
	usages := map[peerPipe]pipeUsage{}

	for _, ourEnd := range ourEnds {
		for peerPid, theirEnds := range endsByPid {
			for _, theirEnd := range theirEnds {
				if !arePipeEnds(ourEnd, theirEnd) {
					continue
				}

				if peerPid == proc.Pid && !isTheReportingEnd(ourEnd, theirEnd) {
					continue
				}

				key := peerPipe{peerPid: peerPid, pipe: pipeIdentity(ourEnd)}

				usage := usages[key]
				usage.canWrite = usage.canWrite || canWrite(ourEnd)
				usage.canRead = usage.canRead || canRead(ourEnd)
				usages[key] = usage
			}
		}
	}

	// How many pipes each line stands for. Two pipes to one peer that we use the
	// same way are one line counting two; used opposite ways they are two lines.
	pipeCounts := map[Connection]int{}
	for key, usage := range usages {
		connection := Connection{
			Peer:      Peer{Name: names[key.peerPid], Pid: key.peerPid},
			Protocol:  ProtocolPipe,
			Direction: pipeDirection(usage),
		}

		pipeCounts[connection]++
	}

	connections := make([]Connection, 0, len(pipeCounts))
	for connection, count := range pipeCounts {
		connection.Count = count
		connections = append(connections, connection)
	}

	SortConnections(connections)

	return connections
}

// One pipe shared with one process, which is what a line stands for before the
// lines to a peer are aggregated into one.
type peerPipe struct {
	peerPid int

	// pipeIdentity() of the pipe
	pipe string
}

// What our own ends of one pipe, taken together, let us do with it.
//
// Both false for a pipe lsof reports no access mode for, which is every anonymous
// pipe on macOS.
type pipeUsage struct {
	canWrite bool
	canRead  bool
}

// True if ours and theirs are two ends of one and the same pipe, so that what is
// written into one can be read out of the other.
//
// Two mechanisms in one predicate, needing no GOOS switch because only one of
// them can fire for any given end. The device clause tests PeerDevice, which is
// populated from an "n->0x..." name and nothing else, and only macOS names an
// anonymous pipe end that way. Such an end carries no inode, so it cannot reach
// the inode clause; every other end can, Linux identifying an anonymous pipe by
// its inode and a named FIFO carrying one on both platforms.
//
// Each clause tests a condition the other platform cannot meet, measured rather
// than assumed. On a quiet macOS laptop 358 of 358 PIPE records carried a
// lowercase "d" device and not one carried an inode, so no macOS anonymous pipe
// reaches the inode clause. In a Debian container all 20 FIFO records carried an
// inode and not one was named "->...", Linux spelling an anonymous pipe "npipe"
// and a named FIFO by its path, so no Linux pipe reaches the device clause.
//
// That first count is about the "d" field and does not carry over to the "->..."
// name, which is the narrower of the two: on the same laptop later, 311 of 311
// PIPE records carried a "d" but only 303 were named "->...". The other 8 are
// pipes whose peer is gone, and they match nothing and get no line, which is what
// PipeConnections() does with any pipe it can find no peer for.
//
// Do not swap this predicate for px's four index maps (px_ipc_map.py:191-220).
// They exist to make its _get_other_end_pids() O(1) per file because Python makes
// the scan expensive, while matching a couple of dozen of our own ends against a
// few thousand pipe files is microseconds in Go. The indexes are also what forces
// px's platform switch: a map key has to be one string, so its fifo_id() must
// choose inode-or-name up front, where a predicate can just test both. px says
// outright that a Linux pipe's name identifies nothing, at px_file.py:85-88.
//
// Testing PeerDevice rather than Device is not only about which platform reported
// the end, which either field would settle. A macOS pipe keeps its Device once its
// peer is gone, and then there is no other end left to name and nothing here for
// it to match.
//
// The inode way needs the file system device as well, two named FIFOs on
// different file systems being free to share an inode number — a FIFO on each of
// two fresh tmpfs mounts is inode 2 on both. And it needs the access modes, since
// every end of a pipe shares its inode and two processes writing into one pipe are
// not talking to each other. The device way needs neither, an end named that way
// pointing at exactly one other end.
//
// Both of those extra tests were watched keeping something out, on Linux, against
// three shells holding FIFO ends on two such tmpfs mounts plus one real pipeline.
// Two shells holding an end of each FIFO reported one another as "pipe (×2)", and
// a shell holding one FIFO for writing on one descriptor and for both on another
// drew one line per peer plus a line to itself. Matching on the inode alone made
// those same pages read "pipe" with no count and five lines instead of three, two
// of them arrows the read-write end contradicts.
//
// This is no check that the two ends are distinct: an end open for reading and
// writing both satisfies it against itself. A caller walking one process' own
// ends has to exclude that, which PipeConnections() does via isTheReportingEnd().
func arePipeEnds(ours PipeEnd, theirs PipeEnd) bool {
	if ours.PeerDevice != "" && ours.PeerDevice == theirs.Device {
		return true
	}

	if ours.Inode == "" || ours.Inode != theirs.Inode {
		return false
	}

	if ours.FileSystemDevice != theirs.FileSystemDevice {
		return false
	}

	// An end that can write pairs with an end that can read, which makes an end
	// open for both a peer of readers, writers and other such ends alike. An end
	// lsof reports no mode for can do neither as far as we know, and pairs with
	// nothing.
	return canWrite(ours) && canRead(theirs) || canRead(ours) && canWrite(theirs)
}

// Which way the data goes through the ends we hold of one pipe: out of us if all
// they let us do is write into it, into us if all they let us do is read it.
//
// DirectionUnknown where they let us do both, since then an arrow either way is a
// claim the other direction contradicts, and where they let us do neither, which
// is what lsof naming no access mode comes to.
func pipeDirection(usage pipeUsage) Direction {
	if usage.canWrite && !usage.canRead {
		return DirectionOutgoing
	}

	if usage.canRead && !usage.canWrite {
		return DirectionIncoming
	}

	return DirectionUnknown
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
// The inode plus the file system it lives on where there is one, that pair being
// the pipe itself — the inode alone would make one pipe of two FIFOs that share an
// inode number on different file systems, and a peer at the end of both would get
// a Count of 1 where it earned 2. Where there is no inode, which is macOS for an
// anonymous pipe, the kernel addresses of the pipe's two ends name it just as
// well, sorted so that either end spells it the same way.
func pipeIdentity(end PipeEnd) string {
	if end.Inode != "" {
		return "inode " + end.FileSystemDevice + " " + end.Inode
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
//
// The file system device is part of it for the same reason arePipeEnds() compares
// it: without it, two ends of two FIFOs that share an inode number are spelled
// identically, and deduplicatePipeEnds() would throw one of them away before
// anything got the chance to match it.
func pipeEndIdentity(end PipeEnd) string {
	return string(end.Access) + "\x00" + end.Device + "\x00" + end.PeerDevice +
		"\x00" + end.FileSystemDevice + "\x00" + end.Inode
}
