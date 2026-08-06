# IPC Connection Visualization — Design

Working notes for IPC connection visualization. **These decisions are settled** —
they came out of a design review, and the rationale for each is recorded below so
it doesn't get relitigated.

**px** is referred to throughout, with file and line references into
`px_ipc_map.py`. It is `walles/px` on GitHub, and is usually checked out next
door at `../px`.

**Lifecycle:** this outlives the implemented slices, because the deferred work at
the bottom depends on it. **Decided: it goes to `main` and stays there** until
pipes and unix domain sockets are both implemented, at which point it dies — and
before deleting it, salvage the "Verified on Linux" findings and the
rejected-alternative rationale into comments next to the code they explain.

Rationale that has already been salvaged is **not repeated here**. The direction
rule and its known limits live in `directionAndPort()`, the deduplication
reasoning in `deduplicateBySocket()`, the whole reverse DNS design in
`resolveAddressesViaDns()`, the one-listing-two-sections decision on
`socketListing`, and the column layout rules in `writeConnectionLines()`. What
remains below is what no code comment carries: measurements, rejected
alternatives, verification findings, and the plan for what isn't built yet.

## Status

**TCP: done.** `GetSocketsByPid()` plus `NetworkConnections()` in
`internal/processes`, rendered by `pageipcconnections.go` and
`pagenetworkconnections.go` over the shared layout in `pageconnections.go`.

**UDP: done.** Same lsof call, same parser, same reversed-pair peer matching —
lsof names a UDP socket in exactly the format it names a TCP one in, so nothing
new was needed for matching and there is no platform specific code. What UDP
added: a `Protocol` field, `DirectionUnknown` rendered `<?>`, and the protocol as
part of the keys identifying a connection. Bound but unconnected UDP sockets are
dropped, see "Deferred" below.

**Pipes: not started.** See "The remaining two kinds".

**Unix domain sockets: not started.** Hardest of the three, and for a reason no
amount of code solves. See "The remaining two kinds".

## The remaining two kinds

There are four kinds of IPC lsof can report: pipes (`PIPE` on macOS, `FIFO` on
Linux), unix domain sockets (`unix`), and network sockets (`IPv4`/`IPv6`) — where
"local" vs "remote" is not a separate detection path, just whether a peer was
found. Network sockets came first because their peer matching is byte-identical
on Linux and macOS.

Both remaining kinds need lsof **without** an `-i` filter, since there is no
filter flag for pipes. That is the 0.27 s / 1.24 MB invocation in the table
below, against 0.13 s / 33 KB for the socket one, and it is a third fork unless
the sections start sharing.

### Pipes

Two mechanisms, verified against real pipe pairs on both platforms:

```
Linux   tFIFO  D0xe  i16466  npipe            ar / aw
macOS   tPIPE  d0x48d4efd2cbb7a037  n->0x7249c1bc766ed78e
```

Linux matches on **file system device plus inode plus opposing `r`/`w` access**;
the name is the literal string `pipe` and identifies nothing, which px says
outright at `px_file.py:85-88`. The device is the uppercase `D` field, and it is
needed because an inode number is only unique within one file system: a FIFO on
each of two fresh tmpfs mounts is inode 2 on both. macOS matches on **our peer's
kernel address against their device**: `theirs.Device ==
strings.TrimPrefix(ours.Name, "->")`.

**These need no `GOOS` switch.** Each clause tests a condition the other platform
cannot meet. Measured on a quiet macOS laptop, 358 of 358 `PIPE` records carry a
lowercase `d` device and **none carries an inode**, so no macOS anonymous pipe
reaches the inode clause; in a Debian container all 20 `FIFO` records carry an
inode and **not one is named `->...`**, Linux spelling an anonymous pipe `npipe`
and a named FIFO by its path, so no Linux pipe reaches the device clause. One
predicate that ORs the two clauses is therefore correct everywhere.

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
while a predicate can just test both.

**Display grammar: reuse `writeConnectionLines()` as it stands.** One line per
pipe, so a process in the middle of a pipeline gets one line per end — the same
shape as every other kind.

Data flows from the writer to the reader, which is a direction worth an arrow and
which maps onto the existing split — write end outgoing, read end incoming, so
`grep(1234) --> sort(5678)` from either end's page. Linux has the access mode
already, since matching needs `a` anyway; wherever it turns out not to be
available, fall back to `DirectionUnknown` and `<?>` — see "Deferred" for where
macOS keeps the access mode when lsof won't hand it over.

Pipes have no port, and `connectionDescription()` appends one unconditionally.
So it needs a no-port case, rendering the bare protocol: `pipe`, and `pipe (×3)`
where several to the same peer aggregate — the aggregation key tolerates a zero
port without changes.

### Unix domain sockets

macOS is nearly free once pipes are done — same `d0x...` device against
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
lsof -n -P -w -iTCP -iUDP -F pfnPT0
```

Measured on a macOS laptop, non-root:

| invocation | time | output |
| --- | --- | --- |
| `lsof -n -P -F fnaptd0iP` (full, px-style) | 0.27 s | 1.24 MB |
| `lsof -n -P -i -F fnaptd0iP` | 0.13 s | 33 KB |
| `lsof -n -P -w -iTCP -F pfnT0` (the TCP slice's, since superseded) | 0.13 s | — |
| `lsof -n -w -d cwd -F pfn0` (already in the tree) | 0.22 s | 15 KB |

This is a **second lsof fork**, separate from the cwd one in `cwds.go`. Rejected
sharing a single full lsof (px's approach) because `-i` scales with socket count
while full lsof scales with every fd on the machine — the difference that matters
for root on a busy multi-user Linux box. It also keeps the two page sections
independently degradable.

**`-iTCP -iUDP` rather than a plain `-i`.** Both protocols in one fork, and lsof
ORs its selection criteria so the two `-i` options add up. Not a bare `-i`: on
macOS that also reports `PICMP` and `PICMPV6` records, which are named `*:*` and
carry nothing we could say anything about. Verified to yield exactly `PTCP` and
`PUDP` on both platforms. (Linux 4.99.4 reports no ICMP sockets under `-i`
anyway, so there the two invocations agree — the narrowing is for macOS.)

**`-Ts` does not narrow the state field down** — verified, the queue sizes come
along anyway. Which is why the parser dispatches on the `ST=` value prefix.

Partial lsof failure is business as usual: use whatever came back, log the rest.
See `cwds.go` for the established handling.

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

**The container recipe** *(run 4)*, `--privileged` being what allows the two
`mount` calls further down:

```
docker run --rm -it --privileged -v "$PWD:/src" -v ftop-gomod:/go/pkg/mod \
  -v ftop-gobuild:/root/.cache/go-build -w /src golang:1.25 bash
# then, inside:
apt-get update && apt-get install -y lsof procps wtmpdb
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
  | sh -s -- -b "$(go env GOPATH)"/bin v2.8.0
wtmpdb boot   # give "last" a database to read, see below
./test.sh
```

The image has neither `lsof` nor the `ps` ftop forks, and `test.sh` wants
golangci-lint v2.8.0, the version `.github/workflows/linux-ci.yml` installs. Two
more things that cost a while to find, both about the container rather than about
ftop. On Debian 13 the `last` that `loginhistory` forks comes from `wtmpdb` rather
than from `util-linux`, and it exits 1 with no database at all, so
`TestGetUsersAtSmoke` needs `wtmpdb boot` once. And run `test.sh` from the
`docker run` shell rather than through a later `docker exec`: an exec'd process has
no parent inside the container's PID namespace, so `TestGetAll`'s walk up to init
lands on the shell instead of on PID 1. With those in place the whole suite is
green.

**The pipe display, exercised by hand** *(run 4)*. Three shells holding FIFO ends
plus one real pipeline, which is the shape the two matching fixes are about:

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

**Both socket sections degrade on their own** *(run 4)*, which is the independence
the two-listing decision was for, observed for the first time. A container with no
internet files at all makes `lsof -iTCP -iUDP` exit 1 with nothing on stdout, so
the socket listing fails outright while the unfiltered pipe listing succeeds: the
IPC section renders `<Unable to list sockets: ...>` and then its pipe lines below
it, and Network Connections renders the error alone.

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

Self-connections (a process dialing its own listening port) are **shown, once** —
deduped by keeping the socket whose local endpoint sorts first. px drops these
entirely (`px_ipc_map.py:173`), which loses information a diagnostic page should
surface. Showing both sides would report `(×2)` for one connection.

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

```
Inter Process Communication
<Detected: TCP, UDP. Not detected: pipes, unix sockets>
curl(999) --> picked(42)                tcp 8080
              picked(42) --> sshd(123)  tcp 22
              picked(42) <?> peer(99)   udp 9001

Network Connections
            picked(42)                     tcp 8080 (listening)
1.2.3.4 --> picked(42)                     tcp 8080 (×12)
            picked(42) --> api.github.com  tcp 443 (×7)
            picked(42) <?> dns.google      udp 53
```

The UDP line in the IPC block needs **both** ends to have connected their
sockets, which is the uncommon shape — see "Deferred" for why a local UDP server
lands in Network Connections under an address instead. Both mockups are drawn to
the real column rules, two spaces between columns and both arrows five columns
wide, so they can be checked against `writeConnectionLines()` rather than trusted.

The arrow direction and the never-an-address rule for the description column are
both stated where they are implemented, in `pageconnections.go`. The two rules
that live nowhere else:

- **Partition on `Peer.Pid != 0`** — IPC section for process peers, Network
  Connections for remote hosts and listeners. A listening socket has no peer at
  all, so it lands in Network, directly above the incoming connections it
  explains.
- The caveat line is **IPC section only** — Network Connections has nothing
  missing. Grow it as kinds land and delete it when nothing is missing.

## Tests

Two conventions that aren't obvious from the repo's other tests:

- Page assertions are **full-block equality after ANSI stripping** (`stripAnsi()`
  and `sectionBody()` in `pagetext_test.go`), not `stringsContains` fragments.
  Alignment is the feature here, and a test asserting `"sshd(123)"` passes whether
  or not the columns line up. The expected string doubles as documentation of what
  the section looks like. `stringsContains` stays for error and empty paths, where
  there is no layout.
- Page sections have `var` seams for **both** the lsof call and the DNS
  resolution. Without a DNS seam every page test does real network lookups.

Parser tests use inline NUL-terminated strings, per `cwds_test.go`. This repo has
no `testdata/` directory and shouldn't grow one for this.

`./test.sh` before any PR, per `AGENTS.md`.

## Deferred, deliberately

- **Pipes** and **unix domain sockets**, both scoped above.
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
- Sharing one lsof invocation across sections, once there are three of them —
  which the pipe work forces the question on, since pipes need an unfiltered lsof.
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
- **A protocol sort key.** Not needed while TCP is the only protocol reaching the
  listening/incoming/outgoing groups and UDP the only one reaching the
  undetermined group, which makes the blocks protocol-pure for free. Pipes and
  unix sockets will break that assumption; revisit then.
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
