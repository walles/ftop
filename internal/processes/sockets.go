package processes

// One TCP socket held open by some process, as reported by lsof.
type Socket struct {
	// lsof's file descriptor number, "31" or similar. Unique within the process
	// holding it, which is what makes it usable for recognizing the same socket
	// reported to us twice.
	Fd string

	// Our own end of the socket: "127.0.0.1:8080", "*:8080" for a listener
	// bound to every interface, or "[::1]:8081" for IPv6.
	Local string

	// The other end, in the same format as Local. Empty for a socket that has no
	// other end: one that is listening, or one that is bound but was never
	// connected.
	Remote string

	Listening bool
}

// Maps PIDs to the TCP sockets held open by the corresponding processes.
//
// The listing is partial: processes we aren't allowed to inspect are missing
// from the map, so expect only a fraction of the running processes when not
// running as root. Processes without any TCP sockets are missing as well, which
// is most of them.
//
// UDP sockets are not included, and neither are pipes or unix domain sockets.
//
// This forks lsof, which takes a fraction of a second. Too slow for calling
// once per frame, fine for on-demand lookups.
func GetSocketsByPid() (map[int][]Socket, error) {
	// TODO: Fork lsof and feed its output to the parser below, the way
	// GetCwdsByPid() does
	parser := newLsofSocketParser()

	return parser.socketsByPid, nil
}

// Parses the output of "lsof -n -P -w -iTCP -F pfnT0", which comes in NUL
// terminated fields, one line per process and then one line per socket:
//
//	p7619\0
//	f31\0n192.168.50.32:57759->172.217.19.234:443\0TST=ESTABLISHED\0TQR=0\0TQS=0\0
type lsofSocketParser struct {
	socketsByPid map[int][]Socket

	// PID from the most recent "p" field, or -1 before the first one
	pid int
}

func newLsofSocketParser() lsofSocketParser {
	return lsofSocketParser{
		socketsByPid: map[int][]Socket{},
		pid:          -1,
	}
}

// Note that lsof escapes non-printable characters, newlines included, so one
// line of output is always one complete record.
func (parser *lsofSocketParser) parseLine(line string) error {
	// TODO: Parse the line into parser.socketsByPid
	return nil
}
