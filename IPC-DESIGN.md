# IPC Connection Visualization — Design

Working notes for IPC connection visualization. **These decisions are settled** —
they came out of a design review, and the rationale for each is recorded below so
it doesn't get relitigated.

**px** is referred to throughout, with file and line references into
`px_ipc_map.py`. It is `walles/px` on GitHub, and is usually checked out next
door at `../px`.

**Lifecycle:** this outlives the implemented slices, because the deferred work at
the bottom depends on it. **Decided: it goes to `main` and stays there** until unix
domain sockets are implemented, at which point it dies — and before deleting it,
salvage the "Verified on Linux" findings and the rejected-alternative rationale
into comments next to the code they explain. Each slice that lands should be
taking its own share of that with it, leaving less to salvage at the end.

Rationale that has already been salvaged is **not repeated here**. The direction
rule and its known limits live in `directionAndPort()`, the deduplication
reasoning in `deduplicateBySocket()`, the whole reverse DNS design in
`resolveAddressesViaDns()`, the one-listing-two-sections decision on
`socketListing`, and the column layout rules in `writeConnectionLines()`. What
remains below is what no code comment carries: measurements, rejected
alternatives, verification findings, and the plan for what isn't built yet.

## Status

**TCP, UDP and pipes: done.** `GetSocketsByPid()` and `GetPipeEndsByPid()` collect,
`NetworkConnections()` and `PipeConnections()` aggregate, all in
`internal/processes`; `pageipcconnections.go` and `pagenetworkconnections.go`
render over the shared layout in `pageconnections.go`. What each kind added is in
git rather than here. Two behaviours to know before reading a page and concluding
it is broken: a bound but unconnected UDP socket gets no line, and an anonymous
pipe on macOS gets no arrow. Both are under "Deferred".

**Unix domain sockets: not started.** Hardest of the three, and for a reason no
amount of code solves. See "Pipes and unix domain sockets".

## Pipes and unix domain sockets

There are four kinds of IPC lsof can report: pipes (`PIPE` on macOS, `FIFO` on
Linux), unix domain sockets (`unix`), and network sockets (`IPv4`/`IPv6`) — where
"local" vs "remote" is not a separate detection path, just whether a peer was
found. Network sockets came first because their peer matching is byte-identical
on Linux and macOS. Unix domain sockets are the one kind still missing; what
follows is how pipes work, and then the plan for those.

Pipes needed lsof **without** an `-i` filter, there being no filter flag for
pipes: the 0.27 s / 1.24 MB invocation in the table below, against 0.13 s / 33 KB
for the socket one, and a third fork unless the sections start sharing. Unix
domain sockets will not force that on us again — `-U` selects them — though they
could ride the pipe listing rather than fork a fourth time. See "Deferred" for
what sharing one fork would cost.

### Pipes

Both matching mechanisms, and the reason one predicate can OR them with no `GOOS`
switch, are documented at `arePipeEnds()`; the lsof records they read are in
`lsofPipeParser`'s doc comment. What no code comment carries is the evidence for
that reason, and the px alternative that was rejected.

**Why the two clauses cannot both fire.** Each tests a condition the other
platform cannot meet. Measured on a quiet macOS laptop, 358 of 358 `PIPE` records
carry a lowercase `d` device and **none carries an inode**, so no macOS anonymous
pipe reaches the inode clause; in a Debian container all 20 `FIFO` records carry an
inode and **not one is named `->...`**, Linux spelling an anonymous pipe `npipe`
and a named FIFO by its path, so no Linux pipe reaches the device clause.

That first measurement is about the `d` field and does not carry over to the
`->...` name, which is the narrower of the two. Measured on the same laptop later:
311 of 311 `PIPE` records carry a `d`, but only 303 are named `->...` — the other
8 are pipes whose peer is gone. Those match nothing and get no line, which is what
`PipeConnections()` does with any pipe it can find no peer for.

The two *device* fields are different things, and neither is what the disjointness
rests on. Lowercase `d` is empty for every `FIFO` record on both platforms — that
is what the earlier "0 of 20 `FIFO` records carry a device" measurement really
established — while uppercase `D` is a file system device Linux reports for every
pipe, anonymous ones living on pipefs and sharing `0xe`. macOS reports no `D` for
a pipe of either kind, so there two pipes are told apart by their inodes and
kernel addresses alone.

**Do not copy px's four index maps** (`px_ipc_map.py:191-220`). They exist to
make `_get_other_end_pids()` O(1) per file because Python makes the scan
expensive; matching ~20 of our own fds against a few thousand pipe files is
microseconds in Go. The indexes are also what *forces* px's platform switch: a
map key has to be one string, so `fifo_id()` must choose inode-or-name up front,
while a predicate can just test both. px also says outright that a Linux pipe's
name identifies nothing, at `px_file.py:85-88`.

### Unix domain sockets

macOS is nearly free now that pipes are done — same `d0x...` device against
`n->0x...` peer scheme, plus a path for listeners.

**Linux is a data problem, not a code problem.** lsof emits nothing to join a
connected pair on. From a real connected pair in a Debian container:

```
server side   f4  d0x000000005e76eec1  i13448  n/tmp/probe.sock type=STREAM
client side   f4  d0x00000000dc23b55d  i16570  ntype=STREAM
```

Different inodes, different devices, no cross-reference, and the client's
connected socket carries **no path at all**. px's Linux fallback — device_number
to files-with-the-same-name, `px_ipc_map.py:260-267` — can only relate processes
sharing a path, which the client end doesn't have, so those files land in px's
`UNKNOWN destinations: Running with sudo might help` bucket
(`px_ipc_map.py:152-153`).

The peer information exists, but only via `ss`, which reports the peer's inode:

```
u_str ESTAB /tmp/probe.sock 10610    * 16721     <- local inode, peer inode
```

So doing this properly on Linux means a **second data source** and inode-keyed
joining, with lsof still carrying the macOS path. Budget for a new collector, not
a new matching function.

## Data collection

```
lsof -n -P -w -i -F pfnPT0
```

Measured on a macOS laptop, non-root:

| invocation | time | output |
| --- | --- | --- |
| `lsof -n -P -F fnaptd0iP` (full, px-style) | 0.27 s | 1.24 MB |
| `lsof -n -P -i -F fnaptd0iP` | 0.13 s | 33 KB |
| `lsof -n -P -w -iTCP -F pfnT0` (the TCP slice's, since superseded) | 0.13 s | — |
| `lsof -n -w -d cwd -F pfn0` (already in the tree) | 0.22 s | 15 KB |

This is a **second lsof fork**, separate from the cwd one in `cwds.go`, and pipes
have since added a third. Sharing one is still rejected; the cost argument is in
`GetSocketsByPid()` and the independent-degradation one on `socketListing`, while
"Deferred" carries the current terms, which the pipe fork changed.

Two measurements behind the flags `sockets.go` documents. **A plain `-i`**, which
is a reversal: the narrower `-iTCP -iUDP` came first, for excluding the `PICMP`
and `PICMPV6` records macOS adds under a bare `-i`, and it cost more than it
bought. Each `-i` is a *search item*, and lsof exits 1 for every item that located
nothing however well the others did, so the pair failed on any machine holding no
socket of one kind — a container with an empty `/proc/net/tcp` failed it always.
With one UDP socket up and no TCP, lsof 4.99.4 prints the socket and still exits
1, which `-V` spells out:

```
bash 4259 root 3u IPv4 28300 0t0 UDP 127.0.0.1:44892->127.0.0.1:9999
lsof: Internet address not located: TCP
```

One item makes a non-zero exit mean "no internet sockets at all", and the ICMP
records get dropped in `parseLine()` instead. An fd selection like `-d cwd` is not
a search item and never exited this way, which is why `cwds.go` and the pipe fork
never saw it. **`-Ts` does not narrow the state field down**, also verified — the
queue sizes come along regardless, which is why the parser dispatches on the `ST=`
value prefix.

Partial lsof failure is business as usual: use whatever came back, log the rest.
See `cwds.go` for the established handling.

**An lsof that ran and exited non-zero is not a failure**, which `sockets.go`
takes further than that established handling. `cwds.go` gives up when a non-zero
exit came with nothing to show, and for sockets that is exactly the idle machine:
nothing to show is the true answer there, and reporting it as
`<Unable to list sockets: ...>` in both page sections is how an empty container
used to render. So the exit status alone no longer fails the listing —
`util.IsExitStatus()` picks that case out, and a command that couldn't be
started, that a signal took down, or whose output wouldn't parse keeps the
established handling. An lsof that isn't installed fails to start, so that one
still says so.

## Verified on Linux

Four runs, all in a container on Debian with **lsof 4.99.4** as root. Runs 1 and
2 used `python:3-slim` (Debian 13.6): the first before the implementation
existed, with real loopback connections — an IPv4 listener on `127.0.0.1:8080`,
an IPv6 listener on `[::1]:8081`, a wildcard listener on `0.0.0.0:8082`, a client
for each, and a client with 8 threads holding one connection — and the second
against the finished TCP code, adding `sshd` (**OpenSSH 10.0p2**) with a live ssh
session, a socket held on two file descriptors, and 400 connections' worth of
load. Run 3 verified UDP, and ran the Go test suite itself in a `golang:1.25`
container against the real Linux lsof. Run 4 did the same for pipes. Findings are
marked with the run they came from where it matters.

Most of what those runs established has since collapsed into the test suite:
reversed-pair matching for both address families and both protocols, IPv6
bracketing, wildcard listeners, `TST=LISTEN` parsing and descriptor
deduplication all have tests now, and run 3 exercised them against real Linux
lsof rather than only against the inline fixtures. What follows is what no test
asserts.

**No per-thread socket duplication** on this lsof, contradicting px's claim at
`px_ipc_map.py:61`: a process with 8 threads holding 1 connection reported
exactly 1 socket. The endpoint-keyed dedup covers that case anyway.

**The `sshd` session child renders as incoming on port 22** *(run 2)* — the case
the machine-wide listen set exists for, observed working for the first time on
either platform. The listener is spelled `*:22`, so port-only wildcard matching
is what ties it to the child's concrete `127.0.0.1:22`, and neither session
process holds a listening socket of its own:

```
pid 985   sshd [listener]          fd 6  n*:22                          LISTEN
pid 985   sshd [listener]          fd 7  n*:22                          LISTEN
pid 996   ssh                      fd 3  n127.0.0.1:60482->127.0.0.1:22
pid 998   sshd-session [priv]      fd 7  n127.0.0.1:22->127.0.0.1:60482
pid 1005  sshd-session root@notty  fd 7  n127.0.0.1:22->127.0.0.1:60482
```

Two more decisions get exercised by that same session. OpenSSH 10 splits the
session into two processes that both hold the accepted socket, which is the
fork-inheritance collision from "Peer matching" in the flesh — lowest PID wins
puts 998 on `ssh`'s line, and it is the same 998 every time the page is opened.
And the listener's two `*:22` sockets, one per address family (Linux spells both
`*:22`, not `[::]:22`), collapse into a single listening row with no count, which
is the dual-stack case on real data rather than on a hand-written test.

**The socket fork earns itself on Linux as well** *(run 2)*, by less than the
rationale hopes: with 819 socket lines against 4122 open files in total, `-iTCP`
took 0.047 s for 55 KB where full lsof took 0.085 s for 180 KB, three runs each
and under 0.02 s of spread. A container understates it — full lsof is the side
that grows with the machine, and this machine had a few dozen processes.

**Non-root degrades instead of failing** *(run 2)*: exit code 0, stderr empty
thanks to `-w`, and exactly one PID reported — our own, its connection the right
way round. Its peer comes back as a bare `127.0.0.1` with no PID, the listening
process being invisible from there.

**The pipe display, exercised by hand** *(run 4)*. The container recipe has moved
to `AGENTS.md`, being useful for verifying anything on Linux rather than pipes in
particular; `--privileged` is what allows the two `mount` calls below. Three shells
holding FIFO ends plus one real pipeline, which is the shape the two matching fixes
are about:

```
mkdir -p /mnt/a /mnt/b
mount -t tmpfs tmpfs /mnt/a && mount -t tmpfs tmpfs /mnt/b
mkfifo /mnt/a/f /mnt/b/f
stat -c 'dev=%D ino=%i %n' /mnt/a/f /mnt/b/f
bash -c 'exec 3<>/mnt/a/f; exec 4<>/mnt/b/f; sleep 3000' &  # an end of each
bash -c 'exec 3</mnt/a/f;  exec 4</mnt/b/f;  sleep 3000' &  # the other ends
bash -c 'exec 3>/mnt/a/f;  exec 4<>/mnt/a/f; sleep 3000' &  # w and u, one FIFO
tail -f /etc/services | sort | nl &                         # a real pipeline
./ftop.sh
```

`stat` reports `dev=37 ino=2` and `dev=38 ino=2`, so the inode collision takes two
`mount` calls rather than any luck — each FIFO is the first file on a file system
that numbers from scratch. Open the shells' pages and the two holding an end of
each FIFO report one another as `pipe (×2)`; the `w`-and-`u` shell draws one line
per peer, `<?>` because its own two ends let it both write the pipe and read it,
plus a line to itself for the FIFO it can write on fd 3 and read back on fd 4. On
the inode alone those pages read `pipe` with no count and five lines instead of
three, two of them arrows the `u` end contradicts, which is what the two fixes
close.

The pipeline gets its arrows, `tail(7612) --> sort(7613)` and
`sort(7613) --> nl(7614)`, Linux reporting the access modes macOS won't — the same
pipeline that reads `<?>` on a laptop, see "Deferred" on taking that direction from
the kernel instead.

**Both socket sections degraded on their own** *(run 4)*, which is the independence
the two-listing decision was for, observed for the only time so far. A container
with no TCP socket made `lsof -iTCP -iUDP` exit 1, so the socket listing failed
outright while the unfiltered pipe listing succeeded: the IPC section rendered
`<Unable to list sockets: ...>` and then its pipe lines below it, and Network
Connections rendered the error alone.

That trigger is gone — an empty machine is no longer a failed listing, see "Data
collection" — and it was the only one anybody had found, so the independence is
back to being designed-for rather than demonstrated.

Still unverified: behaviour on a busy multi-user box, which is the environment
this is ultimately for. Run 2 loaded the container up with sockets and open
files, but a container has a handful of processes and a single user.

## Model — `internal/processes`

**One uniform `Connection` type behind both page sections** (`sockets.go`,
`networkconnections.go`), so merging the two sections later, or splitting them
differently, is a change to the render-time partition and nothing else. Splitting
the model to match the sections is the trap — it would make merging mean unifying
two types, two sorts and two alignment schemes.

## Peer matching

Do **not** copy px's `_local_endpoint_to_pid` map (`px_ipc_map.py:206`), which
maps a single local endpoint to a single pid and lets later entries overwrite
earlier ones. Endpoints are shared in practice — a listener plus every forked
child that accepted on it, and dual-stack listeners. Measured on a quiet laptop:
`2 n*:7000`, `2 n*:5000`. On a multi-worker server as root, px misattributes
connections to whichever process it parsed last. Hence the reversed endpoint pair
key, plus lowest-PID-wins for the fork-inheritance collision.

Known limit, which the px approach doesn't solve either: the two ends may render
the same interface differently (`127.0.0.1` vs `::ffff:127.0.0.1`).

## Direction

The rule, its rationale and its three known limits are all in
`directionAndPort()`; the one-shot-listener limit it names in passing was
reproduced with GNU netcat 0.7.1, which does hold only the accepted socket. One
decision the code doesn't carry:

Self-connections (a process dialing its own listening port) are **shown, once**;
`isTheFarEndOfOurOwnConnection()` says which of the two sockets stands for it. px
drops these entirely (`px_ipc_map.py:173`), which loses information a diagnostic
page should surface, and showing both sides would report `(×2)` for one connection.

## Aggregation

Aggregate by (protocol, direction, peer, port), carrying a `Count`.

**Not optional.** Measured on a quiet laptop, non-root: one process had **252
connections, all to the same peer endpoint**. px renders that as 252 identical
lines. A count is strictly more informative, and it means 252 connections are one
DNS lookup.

The dedup that runs first is keyed on the socket's own identity rather than on
its file descriptor — see `deduplicateBySocket()` for why. lsof's `DEVICE` column
would identify a socket just as well, but it needs another `-F` field and the
endpoints already settle it.

No line cap. After aggregation the pathological case is repetition, not variety,
and the output goes into a pager that handles long content. Add a cap when a
machine demands it.

## Rendering — `internal/ftop`

One file per section, matching the convention from commit 6bcd409, plus
`pageconnections.go` for the column layout the two sections share — one function
taking a `peerLabel` callback rather than the same code twice. The sections keep
rendering their own error and empty states, which is the independence that
mattered. The layout rules are all in `writeConnectionLines()`.

What the two sections look like is no longer drawn here. The page tests assert
whole blocks after ANSI stripping, so their expected strings are the mockups and
cannot rot — `pageipcconnections_test.go` and `pagenetworkconnections_test.go`,
covering both arrows, `<?>`, counts and the listening row. One of them, so that
this section is readable without opening them:

```
<Detected: TCP, UDP, pipes. Not detected: unix sockets>
grep(1234) --> picked(42)                 pipe
               picked(42) --> sort(5678)  pipe
               picked(42) --> sshd(1)     tcp 22
```

A UDP line in the IPC block needs **both** ends to have connected their sockets,
which is the uncommon shape — see "Deferred" for why a local UDP server lands in
Network Connections under an address instead.

The arrow direction and the never-an-address rule for the description column are
both stated where they are implemented, in `pageconnections.go`. The two rules
that live nowhere else:

- **Partition on `Peer.Pid != 0`** — IPC section for process peers, Network
  Connections for remote hosts and listeners. A listening socket has no peer at
  all, so it lands in Network, directly above the incoming connections it
  explains.
- The caveat line is **IPC section only** — Network Connections has nothing
  missing. Grow it as kinds land and delete it when nothing is missing.

## Deferred, deliberately

- **Unix domain sockets**, scoped above. The last of the four kinds, and the only
  thing keeping this document alive.
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
- **Sharing one lsof invocation across sections.** There are three forks now, and
  the pipe one already subsumes the other two: `lsof -n -w -F pfatdDin0` lists
  every open file of every process, so the cwd listing in `cwds.go` and the `-i`
  one in `sockets.go` both ask for subsets of it with different `-F` fields.
  Merging means one call with the union of the fields and three parsers over it,
  and it trades away what "Data collection" above wants kept: `-i` scales with
  socket count where the unfiltered listing scales with every fd on the machine,
  and the sections stop degrading independently — which run 4 observed them doing,
  though nothing triggers that any more. So this is a measurement to make on a
  busy box, not a cleanup to do.
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
- **Pipe direction on macOS, taken from the kernel rather than from lsof.** macOS
  lsof reports no access mode for a `PIPE` record, so every anonymous pipe there
  is `DirectionUnknown` and renders `<?>`. `tail -f /etc/services | sort | nl`
  shows as `tail(40504) <?> sort(40505)  pipe` when the truth is plainly
  `tail --> sort`.

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
