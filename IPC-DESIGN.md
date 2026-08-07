# IPC Connection Visualization — Design

Working notes for IPC connection visualization. **These decisions are settled** —
they came out of a design review, and the rationale for each is recorded below so
it doesn't get relitigated.

**px** is referred to throughout, with file and line references into
`px_ipc_map.py`. It is `walles/px` on GitHub, and is usually checked out next
door at `../px`.

**Lifecycle:** this outlives the implemented slices, because the deferred work at
the bottom depends on it. **Decided: it goes to `main` and stays there** until unix
domain sockets work on Linux too, at which point it dies — and before deleting it,
salvage the "Verified on Linux" findings and the rejected-alternative rationale
into comments next to the code they explain. Each slice that lands should be
taking its own share of that with it, leaving less to salvage at the end.

Rationale that has already been salvaged is **not repeated here**. The direction
rule and its known limits live in `directionAndPort()`, the deduplication
reasoning in `deduplicateBySocket()`, the whole reverse DNS design in
`resolveAddressesViaDns()`, the one-listing-two-sections decision on
`socketListing`, and the column layout rules in `writeConnectionLines()`. The
macOS unix socket slice took its own share the same way: the direction rule and
the measurements behind it are in `unixSocketDirection()`, the record shapes in
`lsofUnixSocketParser` and `applyUnixSocketName()`, lowest-PID-wins in
`unixSocketsByDevice()`, why devices compare as strings on `UnixSocket.Device`,
the flags and the exit status in `GetUnixSocketsByPid()`, and why it is a listing
of its own on `unixSocketListing`.

What remains below is what no code comment carries: measurements, rejected
alternatives, verification findings, and the plan for what isn't built yet.

## Status

**TCP, UDP and pipes: done on both platforms. Unix domain sockets: done on
macOS.** `GetSocketsByPid()`,
`GetPipeEndsByPid()` and `GetUnixSocketsByPid()` collect, `NetworkConnections()`,
`PipeConnections()` and `UnixSocketConnections()` aggregate, all in
`internal/processes`; `pageipcconnections.go` and `pagenetworkconnections.go`
render over the shared layout in `pageconnections.go`. What each kind added is in
git rather than here. Three behaviours to know before reading a page and
concluding it is broken: a bound but unconnected UDP socket gets no line, an
anonymous pipe on macOS gets no arrow, and a unix socket nobody has dialed gets
no line either. All three are under "Deferred".

**Unix domain sockets on Linux: not started**, and the last thing keeping this
document alive. lsof there reports a unix socket's own kernel address and no
peer, so the peer edge has to come from netlink instead. See "Unix domain
sockets".

## Choosing a data source

lsof is a means, not the design. Pick the source per kind of IPC and per
platform, and let them differ: lsof where it is the best fit, netlink where it
carries an edge lsof cannot see, `proc_pidfdinfo` where the kernel knows
something lsof drops. Two of the plans below already leave lsof for exactly
those reasons.

**Fork count is not a design constraint.** Don't fork inside a loop — that is
the shape that scales with the machine. Beyond that, fork as many times as the
job needs, and fix it if a measurement says it hurts. Where the text below
argues from "one fewer fork", read it as a tiebreaker between otherwise equal
options, never as a reason to reuse a listing that fits the job worse.

## Pipes and unix domain sockets

There are four kinds of IPC this page covers: pipes (`PIPE` on macOS, `FIFO` on
Linux), unix domain sockets (`unix`), and network sockets (`IPv4`/`IPv6`) — the
spellings being lsof's, which is today's source for all of them but not what
defines the list. "Local" vs "remote" is not a separate detection path, just
whether a peer was found. Network sockets came first because their peer matching is byte-identical
on Linux and macOS. Unix domain sockets came last, and are the one kind whose two
platforms need two different sources; what follows is how pipes work, and then
what Linux still needs.

Pipes needed lsof **without** an `-i` filter, there being no filter flag for
pipes: the 0.27 s / 1.24 MB invocation in the table below, against 0.13 s / 33 KB
for the socket one, and a third fork. `-U` spared unix sockets that, and why they
got a listing of their own rather than riding the pipe one is on
`unixSocketListing`. Sharing one listing is rejected at the bottom.

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
rests on. Lowercase `d` is empty for every `FIFO` record on both platforms, while
uppercase `D` is a file system device Linux reports for every pipe, anonymous ones
living on pipefs and sharing `0xe`. macOS reports no `D` for a pipe of either
kind, so there two pipes are told apart by their inodes and kernel addresses
alone.

**Do not copy px's four index maps** (`px_ipc_map.py:191-220`). They exist to
make `_get_other_end_pids()` O(1) per file because Python makes the scan
expensive; matching ~20 of our own fds against a few thousand pipe files is
microseconds in Go. The indexes are also what *forces* px's platform switch: a
map key has to be one string, so `fifo_id()` must choose inode-or-name up front,
while a predicate can just test both. px also says outright that a Linux pipe's
name identifies nothing, at `px_file.py:85-88`.

### Unix domain sockets

**macOS is done, Linux is the slice that is left.** macOS went first because it
was a matching and rendering change over the pipe machinery with no new
collector, on the platform where `./test.sh` runs and the page can be looked at,
and because it settled the model questions — whether a connection gets a path
above all — before the netlink collector forced iterating on the render layer
through a container.

What the macOS half established is in the code, pointed at from the top of this
document. Two of its decisions bind the Linux slice as well:

- **A connection is its two sockets, labelled by the path it was made over.** So
  whatever Linux reads, it has to come up with the same two facts per socket —
  who is at the other end, and the path where there is one.
- **A socket nobody has dialed gets no line**, listeners included, which is the
  partition rule rather than a macOS limitation. See "Deferred".

#### Linux: a netlink collector, not `ss`

**`/proc` alone cannot do it.** Verified against a real connected pair plus a
listener in a `golang:1.25` container. `/proc/net/unix` has no peer column, and
its `Num` field is the socket's own kernel address — the same number lsof prints
as `d0x...`, so lsof's device field is this file reformatted and neither source
knows a peer:

```
Num       RefCount Protocol Flags    Type St Inode Path
00000000609cd1bd: 00000003 00000000 00000000 0001 03   378                   <- client end
000000001942c573: 00000003 00000000 00000000 0001 03   379 /tmp/probe.sock   <- accepted end
00000000ee2a9915: 00000002 00000000 00010000 0001 01   369 /tmp/probe.sock   <- listener
```

The client's connected socket carries **no path at all**, which is what sinks
px's Linux fallback — device_number to files-with-the-same-name,
`px_ipc_map.py:260-267`, can only relate processes sharing a path. Those files
land in px's `UNKNOWN destinations: Running with sudo might help` bucket
(`px_ipc_map.py:152-153`).

**Take the peer from `sock_diag` netlink, not by forking `ss`.** `ss -x` reports
nothing beyond what a `UNIX_DIAG_SHOW_PEER` dump carries, and asking for that
dump ourselves is around 60 lines. Same container, non-root:

```
ino=378   state=1  peer=379   name=""                   <- client end
ino=379   state=1  peer=378   name="/tmp/probe.sock"    <- accepted end
ino=9391  state=1  peer=381   name=""
ino=381   state=1  peer=9391  name="/tmp/probe.sock"
ino=369   state=10 peer=0     name="/tmp/probe.sock"    <- listener, no peer
```

Why netlink, in the order that decides it:

- **Nothing else has the peer.** lsof and `/proc/net/unix` both report a socket's
  own kernel address and stop there, and `ss -x` gets its peer column from this
  same dump. Netlink is not a cheaper way to the same data, it is the only way to
  the edge.
- **The pairing is exact and symmetric**, so joining is a lookup with no
  tiebreaking — none of the lowest-PID-wins that fork inheritance forces on TCP.
- **It works non-root**, where lsof degrades: re-running the dump as `nobody`
  gave byte-identical output, peers included. That was a container, so how it
  behaves against another user's sockets on a real box is reasoning rather than a
  measurement.
- **Netlink carries no PID.** `ss -p` gets those by walking `/proc/*/fd` itself.
  So lsof stays the pid/fd/inode source and netlink supplies only the peer edge,
  keyed on inode. Inodes matched across `/proc/net/unix`, `ss -x`, `lsof -U` and
  the netlink dump in that run.

Cheap on top of that, but not why: `golang.org/x/sys/unix` is already required
and `syscall.ParseNetlinkMessage` is stdlib, so this is a socket and a parse
rather than another `exec`.

**Why lsof and not `/proc/*/fd` for the pid/fd/inode side.** Reading the fd
symlinks ourselves gives the same join — they spell out `socket:[378]` — with no
fork at all. lsof keeps the job because the record shape and the parser are then
shared with macOS, which has no `/proc` to walk; a Linux-only collector would be
a second code path for the half of the data both platforms already agree on.
Revisit if lsof turns out to be the slow part, not to save the fork.

**The platform asymmetry to design for: none, as it turned out.** This used to
say that the netlink name gives an accepted socket its path on Linux while macOS
leaves it empty, so the path had to be optional with macOS the platform doing
without. The macOS half was wrong: an accepted socket there is named by its path
too, which is what `UnixSocket.Path` now documents. Both platforms fill the path
in the same way, and what leaves it empty is a `socketpair(2)`, on either of them.

**Direction is an open question here**, and the one part of the model macOS did
not settle. The rule there is that the end naming the other is the one that
dialed, see `unixSocketDirection()`, and it cannot carry over: a netlink peer edge
is symmetric, so both ends of every pair name each other and every pair Linux can
see both ends of reads `<?>`. The same fact is available in another shape — in the
dump above the client's end has `name=""` while the accepted end carries the path,
so **the end without a path is the one that dialed**, and a `socketpair(2)` has
neither.

macOS corroborates that rule rather than contradicting it: its 29 client records
carry no path while the socket each of them names does, and a `socketpair(2)` there
has no path on either end. So Linux would be reading the same fact off the path
instead of off who names whom. Still only one observed Linux pair, so verify it
before building on it.

**lsof's Linux name field is not a bare path.** Verified in a `golang:1.25`
container with lsof 4.99.4 against a real connected pair, where
`lsof -n -w -U -F pfnd0` reports:

```
f4\0d0x00000000185fc59b\0n/tmp/probe.sock type=STREAM\0   <- listener
f5\0d0x000000001e605e5a\0ntype=STREAM\0                   <- client end
f8\0d0x00000000752931c2\0n/tmp/probe.sock type=STREAM\0   <- accepted end
```

A ` type=STREAM` suffix on every name, and the client end named by nothing else at
all — no peer anywhere, as the rest of this section says. So the path comes from
netlink, or from this with the suffix cut off — which `applyUnixSocketName()`
knowingly does not do. Until then `GetUnixSocketsByPid()` does work on Linux and
returns sockets with that suffix in `Path` and `PeerDevice` empty on all of them,
so `UnixSocketConnections()` matches no pair and contributes no line to the IPC
section. Pipes and TCP still fill it. That is the deferred state and not a bug.

## Data collection

```
lsof -n -P -w -i -F pfnPT0
```

Measured on a macOS laptop, non-root:

| invocation | time | output |
| --- | --- | --- |
| `lsof -n -P -F fnaptd0iP` (full, px-style) | 0.27 s | 1.24 MB |
| `lsof -n -P -i -F fnaptd0iP` | 0.13 s | 33 KB |
| `lsof -n -w -d cwd -F pfn0` (already in the tree) | 0.22 s | 15 KB |

This is a **second lsof fork**, separate from the cwd one in `cwds.go`; pipes have
since added a third and unix sockets a fourth. Sharing one is still rejected on
the terms the "Deferred" bullet on sharing carries: what `-i` costs against an
unfiltered listing, and the sections' independent degradation.

**`-U` costs what `-i` costs**, which is what `GetUnixSocketsByPid()` claims and
this is the measurement behind it. A later run on the same laptop, median of
three, timing the invocations actually in the tree rather than the px-style ones
above:

| invocation | time | output |
| --- | --- | --- |
| `lsof -n -w -U -F pfnd0` (unix sockets) | 0.17 s | 26 KB |
| `lsof -n -P -w -i -F pfnPT0` (network sockets) | 0.18 s | 34 KB |
| `lsof -n -w -F pfatdDin0` (pipes, unfiltered) | 0.39 s | 1.4 MB |

So the fourth fork is a fifth of the third one's cost, and the pipe listing
remains the expensive one — the reason `-U` got an invocation of its own rather
than riding it.

Two measurements behind the flags `sockets.go` documents. **Why a plain `-i` and
not `-iTCP -iUDP`**, which is the narrower spelling and would keep out the `PICMP`
and `PICMPV6` records macOS adds under a bare `-i`: the pair costs more than it
buys. Each `-i` is a *search item*, and lsof exits 1 for every item that located
nothing however well the others did, so the pair fails on any machine holding no
socket of one kind — a container with an empty `/proc/net/tcp` fails it always.
With one UDP socket up and no TCP, lsof 4.99.4 prints the socket and still exits
1, which `-V` spells out:

```
bash 4259 root 3u IPv4 28300 0t0 UDP 127.0.0.1:44892->127.0.0.1:9999
lsof: Internet address not located: TCP
```

One item makes a non-zero exit mean "no internet sockets at all", and the ICMP
records get dropped in `parseLine()` instead. An fd selection like `-d cwd` is not
a search item and never exits this way, which is why `cwds.go` and the pipe fork
never see it. **`-Ts` does not narrow the state field down**, also verified — the
queue sizes come along regardless, which is why the parser dispatches on the `ST=`
value prefix.

Partial lsof failure is business as usual: use whatever came back, log the rest.
See `cwds.go` for the established handling.

**An lsof that ran and exited non-zero is not a failure**, which `sockets.go` and
`unixsockets.go` both take further than that established handling. `cwds.go` gives
up when a non-zero exit came with nothing to show, and for sockets that is exactly
the idle machine:
nothing to show is the true answer there, and reporting it as
`<Unable to list sockets: ...>` in both page sections is how an empty container
used to render. So the exit status alone no longer fails the listing —
`util.IsExitStatus()` picks that case out, and a command that couldn't be
started, that a signal took down, or whose output wouldn't parse keeps the
established handling. An lsof that isn't installed fails to start, so that one
still says so.

## Verified on Linux

Five runs, all in a container on Debian with **lsof 4.99.4**, the first four as
root. Runs 1 and
2 used `python:3-slim` (Debian 13.6): the first before the implementation
existed, with real loopback connections — an IPv4 listener on `127.0.0.1:8080`,
an IPv6 listener on `[::1]:8081`, a wildcard listener on `0.0.0.0:8082`, a client
for each, and a client with 8 threads holding one connection — and the second
against the finished TCP code, adding `sshd` (**OpenSSH 10.0p2**) with a live ssh
session, a socket held on two file descriptors, and 400 connections' worth of
load. Run 3 verified UDP, and ran the Go test suite itself in a `golang:1.25`
container against the real Linux lsof. Run 4 did the same for pipes. Run 5 came
with the macOS unix socket slice: a `golang:1.25` `test.sh` run, where the two real
lsof unix socket tests skip by design, plus the two `-U` findings in the Linux
subsection above. Findings are marked with the run they came from where it matters.

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
holding FIFO ends plus one real pipeline, which is the shape that tells the inode
clause in `arePipeEnds()` apart from a bare inode comparison:

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
plus a line to itself for the FIFO it can write on fd 3 and read back on fd 4.
Matching on the inode alone makes those same pages read `pipe` with no count and
five lines instead of three, two of them arrows the `u` end contradicts — which is
what the file system device check and the read-pairs-with-write test in the inode
clause are each keeping out.

The pipeline gets its arrows, `tail(7612) --> sort(7613)` and
`sort(7613) --> nl(7614)`, Linux reporting the access modes macOS won't — the same
pipeline that reads `<?>` on a laptop, see "Deferred" on taking that direction from
the kernel instead.

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

Aggregate by (protocol, direction, peer, port, path), carrying a `Count`. Why the
path is in that key is on `UnixSocketConnections()`.

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
grep(1234) --> picked(42)                 pipe
               picked(42) --> sort(5678)  pipe
               picked(42) --> sshd(1)     tcp 22
```

A UDP line in the IPC block needs **both** ends to have connected their sockets,
which is the uncommon shape — see "Deferred" for why a local UDP server lands in
Network Connections under an address instead.

The arrow direction and the never-an-address rule for the description column are
both stated where they are implemented, in `pageconnections.go`. The one rule that
lives nowhere else:

- **Partition on `Peer.Pid != 0`** — IPC section for process peers, Network
  Connections for remote hosts and listeners. A listening socket has no peer at
  all, so it lands in Network, directly above the incoming connections it
  explains.

**No line naming the kinds this section covers**, unix sockets included.
`<No connections found>` is true of them too — we found none, having never looked —
and every tool has limits it doesn't recite. A listing that fails still says so,
which is the one gap specific enough to earn a line.

## Deferred, deliberately

- **Unix domain sockets on Linux**, scoped above. The last thing keeping this
  document alive, macOS having landed.
- **Listening unix sockets get no line.** A listener has a path and no peer, and
  the partition rule sends a peerless connection to Network Connections — which
  would file `/tmp/foo.sock` under "Network". Wrong section, and an exception to a
  settled rule was not worth carving out for it in the macOS slice. The cost is
  small: every client actually using the listener already gets a line, so what
  goes missing is only "this process offers a socket nobody is using". Reopen it
  along with whatever eventually decides where a local endpoint that isn't a
  process belongs.
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
  means one call with the union of the fields and four parsers over it. Saving a fork is not a reason to want that —
  see "Choosing a data source" — and it trades away what "Data collection" wants
  kept: `-i` scales with socket count where the unfiltered listing scales with
  every fd on the machine, and the sections stop degrading independently. Only a
  measurement on a busy box, showing the forks themselves are what hurts, would
  reopen it.
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
