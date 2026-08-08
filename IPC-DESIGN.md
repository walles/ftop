# IPC Connection Visualization — What Isn't Built

Working notes for the parts of IPC connection visualization that are **deliberately
not built**. What is built has its rationale in a comment next to the code
instead: the measurements, the rejected alternatives and the verification findings
all moved there as each slice landed, which is what leaves this file down to plans
and open questions.

**Lifecycle:** it lives as long as the list below has anything on it. Whatever
gets picked up from here takes its reasoning into the code the same way, and the
bullet goes.

**px** is cited by file and line from several of those code comments,
`px_ipc_map.py` and `px_file.py`. It is `walles/px` on GitHub, and is usually
checked out next door at `../px`.

## Choosing a data source

lsof is a means, not the design. Pick the source per kind of IPC and per platform,
and let them differ: lsof where it is the best fit, netlink where it carries an
edge lsof cannot see, `proc_pidfdinfo` where the kernel knows something lsof
drops. Unix domain sockets on Linux already left lsof for the second of those
reasons, and pipe direction on macOS for the third.

**Fork count is not a design constraint.** Don't fork inside a loop — that is the
shape that scales with the machine. Beyond that, fork as many times as the job
needs, and fix it if a measurement says it hurts. Where an argument below turns on
"one fewer fork", read it as a tiebreaker between otherwise equal options, never as
a reason to reuse a listing that fits the job worse.

## Still unverified

Behaviour on a busy multi-user box, which is the environment this is ultimately
for. The Linux runs loaded a container up with sockets and open files, but a
container has a handful of processes and a single user. Same for the netlink dump
against another user's sockets: it came back byte identical as `nobody` in a
container, and what that means on a real box is reasoning rather than a
measurement.

## Deferred, deliberately

- **Listening unix sockets get no line.** A listener has a path and no peer, and
  the partition rule sends a peerless connection to Network Connections — which
  would file `/tmp/foo.sock` under "Network". Wrong section, and an exception to a
  settled rule wasn't worth carving out for it. The cost is small: every client
  actually using the listener already gets a line, so what goes missing is only
  "this process offers a socket nobody is using". Reopen it along with whatever
  eventually decides where a local endpoint that isn't a process belongs.
- **Bound but unconnected UDP sockets get no line.** They are dropped along with
  the bound TCP sockets that never carried anything. UDP has no listening state,
  so lsof gives us no way to tell a server's bound socket from the ephemeral
  source port of something that merely called `sendto()`, and labelling either one
  would be a guess.

  This costs more than the missing row, and the second cost is the bigger one:

  1. A process serving UDP and nothing else, holding only `*:53`, shows nothing.
  2. **Its clients can't name it either.** A UDP server serves every client from
     one bound socket and never connects it, so there is no reversed pair to
     match. The client's peer comes back as an address with no PID, which puts it
     in *Network Connections* reading `picked(42) <?> localhost  udp 53` — with
     the server sitting right there in the same lsof listing, unnamed.

  So for UDP the IPC section stays empty unless both ends happen to have
  connected their sockets, which is the uncommon shape. Worth knowing before
  concluding the peer matching is broken.

  The honest fix, when it is worth the code, is a third state rendered
  `udp 53 (bound)` — "bound" being a fact rather than an inference — accepting
  that ephemeral source ports get a line too. Costs a field on `Connection`, a
  render suffix, a sort group and their tests. It would also give clients their
  peer back, since a bound socket in the listing could then be matched on its
  port.
- `(ssh)` service-name annotations next to port numbers. Cosmetic, needs
  `/etc/services` parsing, no model change.
- A line cap for processes with hundreds of *distinct* peers.
- **Sharing one lsof invocation across sections — rejected, not pending.** There
  are four forks now, and the pipe one already subsumes the other three:
  `lsof -n -w -F pfatdDin0` lists every open file of every process, so the cwd
  listing in `cwds.go`, the `-i` one in `sockets.go` and the `-U` one in
  `unixsockets.go` all ask for subsets of it with different `-F` fields. Merging
  means one call with the union of the fields and four parsers over it. Saving a
  fork is not a reason to want that — see "Choosing a data source" — and it trades
  away two things worth keeping: `-i` and `-U` scale with socket count where the
  unfiltered listing scales with every fd on the machine, and the sections stop
  degrading independently. Only a measurement on a busy box, showing the forks
  themselves are what hurts, would reopen it.
- **Re-sorting remote peers by resolved name.** Rows sort on `Peer.Name`, which
  for a remote host is its address, and then render as a host name — so with
  several remote peers the visible order isn't alphabetical by what the reader
  sees. Sorting in the pure layer has nothing else to sort on, and fixing it means
  sorting again in the page after DNS. Considered and left alone.
- **A seam inside `resolveAddressesViaDns`.** Asserting that duplicate addresses
  are only looked up once needs an injectable `LookupAddr`, and the alternative —
  reverse resolving a TEST-NET address for real — puts a network call and up to
  2 s into the test suite. The function's fallback behaviour is covered at page
  level instead, via an address the fake resolver has no answer for.
- **A test for the `listening` half of the socket identity.**
  `deduplicateBySocket()` keys on (protocol, local, remote, listening), and
  replacing that last field with a constant passes the whole suite — verified by
  mutation. Nothing constructs the case the field exists for: a listener and the
  bound but unconnected socket sharing its address, which are two sockets with
  identical protocol, local address and empty remote. Collapse them and whichever
  lsof happened to report first wins, so a listening row disappears from Network
  Connections about half the time. Direction inference elsewhere survives it,
  since `listenSets()` reads the raw listing rather than the deduplicated one.
  Predates UDP; cheap to close.
