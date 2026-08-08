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
reasons, and the pipe direction work below would leave it for the third.

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
- **The no-op darwin half of `fillInPeersAndPaths()` — flagged for a look.**
  `unixsocketpeers_linux.go` takes the peer edge and the path from a netlink dump,
  and `unixsocketpeers_darwin.go` exists only to say there is nothing to do there,
  its whole body being `return nil`. That is a file per platform where the repo's
  other splits — `io_darwin.go`/`io_linux.go`,
  `sysload_darwin.go`/`sysload_linux.go` — have both halves doing real work, and
  all the seam buys is one call in `GetUnixSocketsByPid()`.

  Two alternatives, and one non-alternative. A `//go:build !linux` file would at
  least cover a platform nobody has added yet, where `_darwin.go` leaves the build
  broken for one. Giving each platform its own `GetUnixSocketsByPid()` keeps the
  seam out of the shared code and duplicates the lsof fork and its error handling
  instead. What is out is doing it in the shared file behind a `runtime.GOOS`
  test: the netlink constants, `unix.NlMsghdr` and `syscall.ParseNetlinkMessage`
  don't exist in a darwin build, so the file wouldn't compile.
- **Pipe direction on macOS, taken from the kernel rather than from lsof.** macOS
  lsof reports no access mode for a `PIPE` record, so every anonymous pipe there
  is `DirectionUnknown` and renders `<?>`. `tail -f /etc/services | sort | nl`
  shows as `tail(40504) <?> sort(40505)  pipe` when the truth is plainly
  `tail --> sort`. Linux reports the modes and gets its arrows already.

  **Not a matter of asking lsof for the right field.** `+fg` is lsof's file-flag
  option, and it fills that column in for `CHR` and `REG` while leaving it blank
  for `PIPE`; the FD number carries no `r`/`w` suffix either. Both ends of a real
  pipeline, on macOS lsof 4.91:

  ```
  tail  40697  0r  CHR   R;SH          /dev/null     <- flags reported
  tail  40697  1   PIPE                ->0x8512...   <- blank
  sort  40698  0   PIPE                ->0xe272...   <- blank
  sort  40698  2w  CHR   W,0x10000;SH  /dev/null     <- flags reported
  ```

  **The kernel knows.** `proc_pidfdinfo(pid, fd, PROC_PIDFDPIPEINFO)` fills in a
  `proc_fileinfo` whose `fi_openflags` carries FREAD/FWRITE, verified against
  that same pipeline:

  ```
  tail(40697) fd 1  fi_openflags=0x10002  FWRITE  handle=0xe272559b6a8c5e34  peer=0x8512e4a289d94ded
  sort(40698) fd 0  fi_openflags=0x1      FREAD   handle=0x8512e4a289d94ded  peer=0xe272559b6a8c5e34
  ```

  `pipe_handle` and `pipe_peerhandle` are the same two numbers lsof prints as the
  `DEVICE` and the `->` name, so this is the pipe end we already have with the
  direction attached, not a second identity to join on.

  Cheaper than a new data source usually is. cgo is already a macOS dependency —
  `test.sh` builds both darwin targets with `CGO_ENABLED=1`, and
  `sysload_darwin.go` is the established `//go:build darwin` plus inline C
  pattern. The change lands entirely in `GetPipeEndsByPid()`, filling in the
  `Access` lsof left empty, so `arePipeEnds()`, `pipeDirection()` and every one of
  their tests are untouched. Linux keeps taking the access mode from lsof and
  needs nothing. Cost is one syscall per pipe end, around 300 on a quiet laptop,
  against the 0.36 s the unfiltered lsof already spends.

  Guard the race between the lsof fork and the syscall by requiring the returned
  `pipe_handle` to equal the `Device` lsof reported — an fd can be closed and
  reopened in between, and then the flags describe some other file entirely.

  Unverified: how this degrades for processes we don't own. `proc_pidfdinfo`
  enforces a same-uid-or-root check, so EPERM and a fall back to
  `DirectionUnknown` is what to expect, matching how lsof already degrades — but
  the laptop this was measured on had no root-owned process holding a pipe to
  probe, so that is reasoning rather than a measurement.
