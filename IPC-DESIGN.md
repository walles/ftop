# IPC Connection Visualization — Design

Working notes for IPC connection visualization. **These decisions are settled** —
they came out of a design review, and the rationale for each is recorded below so
it doesn't get relitigated.

**Lifecycle:** this outlives the TCP slice, because the deferred work at the
bottom depends on it. It should die once pipes, unix sockets and UDP are all
implemented — and before deleting it, salvage the "Verified on Linux" findings
and the rejected-alternative rationale into comments next to the code they
explain. Do not merge it to `main` without deciding which of those two things is
happening.

## Scope of this slice

TCP only. Two new page sections on the process info page, replacing the
hand-written mockup lines currently in `internal/ftop/pageipcconnections.go`.

There are four kinds of IPC that lsof can report: pipes (`PIPE` on macOS,
`FIFO` on Linux), unix domain sockets (`unix`), and network sockets
(`IPv4`/`IPv6`) — where "local" vs "remote" is not a separate detection path,
just whether a peer was found. TCP was picked first because its peer matching is
byte-identical on Linux and macOS, while pipes need two distinct code paths
(Linux: inode + opposing `r`/`w` access; macOS: unique names).

UDP is excluded on purpose, see "Direction" below.

## Data collection

```
lsof -n -P -w -iTCP -F pfnT0
```

- `-n` no host resolution (slow, and we do our own)
- `-P` numeric ports; without it lsof yields service names like `ipp` for 631,
  which don't sort numerically
- `-w` no warnings about processes we can't inspect
- `-iTCP` network sockets only, TCP only. Verified to leak zero UDP entries.
- `-F pfnT0` pid, fd, name, TCP state, NUL separated

Measured on a macOS laptop, non-root:

| invocation | time | output |
| --- | --- | --- |
| `lsof -n -P -F fnaptd0iP` (full, px-style) | 0.27 s | 1.24 MB |
| `lsof -n -P -i -F fnaptd0iP` | 0.13 s | 33 KB |
| `lsof -n -P -w -iTCP -F pfnT0` | 0.13 s | — |
| `lsof -n -w -d cwd -F pfn0` (already in the tree) | 0.22 s | 15 KB |

This is a **second lsof fork**, separate from the cwd one in `cwds.go`. Rejected
sharing a single full lsof (px's approach) because `-iTCP` scales with socket
count while full lsof scales with every fd on the machine — the difference that
matters for root on a busy multi-user Linux box. It also keeps the two page
sections independently degradable.

Partial lsof failure is business as usual: use whatever came back, log the rest.
See `cwds.go` for the established handling.

**Parser requirement:** asking for the `T` field yields *three* fields, all
starting with `T` — `TST=ESTABLISHED`, `TQR=0`, `TQS=0`. So dispatch on the value
prefix (`ST=`), not on the type character alone. Ignoring `TQR=`/`TQS=` is what
`lsofCwdParser` already does with every field type it doesn't recognise.
`-Ts` does *not* narrow this down — verified, queue sizes come along anyway.

**One listing, both sections.** `writeProcessInfo` lists the sockets once (behind
a `sync.OnceValue`, so the sections above it still stream while lsof runs) and
passes the result to both sections. Two independent calls would let the sections
disagree about a connection that came or went in between, and the failure would
be silent and rare. It costs each section an argument, and it departs from
`cwdFriendsForPaging` calling `getCwdsByPid()` itself — a convention that only
held while every section had exactly one consumer. Each section still renders its
own error state, which is why the error rides along in the listing.

## Verified on Linux

Run in a `python:3-slim` container (Debian 13.6, **lsof 4.99.4**) with real
loopback connections: an IPv4 listener on `127.0.0.1:8080`, an IPv6 listener on
`[::1]:8081`, a wildcard listener on `0.0.0.0:8082`, a client for each, and a
client with 8 threads holding one connection.

**Reversed-pair matching holds byte-for-byte, both address families.** This is
the central mechanism of the design, and it was previously unverified on either
platform — a quiet macOS laptop has no connected loopback sockets to observe.

```
pid 179 (listener)   n127.0.0.1:8080->127.0.0.1:33102
pid 186 (client)     n127.0.0.1:33102->127.0.0.1:8080     exact reverse

pid 180 (listener)   n[::1]:8081->[::1]:46208
pid 187 (client)     n[::1]:46208->[::1]:8081             exact reverse
```

**IPv6 is bracketed on Linux exactly as on macOS** (`[::1]:8081`), so one
endpoint parser covers both platforms.

**Wildcard listeners behave as the direction rule assumes.** The listener
reports `n*:8082` while the socket it accepted reports
`n127.0.0.1:8082->127.0.0.1:48368` — concrete local address against a wildcard
listener. Port-only matching for `*:`-bound listeners is genuinely required, not
defensive.

**No per-thread socket duplication** on this lsof, see "Aggregation" below.

Still unverified: behaviour as root on a busy multi-user box, which is the
environment this is ultimately for. The container ran as root but with only a
handful of processes.

## Model — `internal/processes`

```go
type Peer struct {
	Name string // command name; remote address when Pid == 0; empty when listening
	Pid  int    // 0 for a remote host
}

type Connection struct {
	Peer      Peer
	Direction Direction // DirectionIncoming | DirectionOutgoing
	Port      int       // the server port, never the client's
	Listening bool
	Count     int
}
```

Files:

- `sockets.go` — `GetSocketsByPid()`, forks lsof and parses. Mirrors `cwds.go`.
- `networkconnections.go` — pure function over the parsed sockets, mirrors
  `cwdfriends.go`. Does the aggregation.

Notes on the shape:

- One uniform `Connection` type behind **both** sections, so merging the two
  sections later (or splitting differently) is a change to the render-time
  partition and nothing else. Splitting the model to match the sections is the
  trap: it would make merging mean unifying two types, two sorts and two
  alignment schemes.
- `Peer.Pid == 0` is the discriminator meaning "`Name` is a remote address".
  That keeps the detection layer pure and fast, with DNS happening in the page.
- `Peer{Name: ""}` renders as `PID 1234`, px-style. Happens when a peer started
  after `allProcesses` was snapshotted, since lsof runs later.
- No `Protocol` field: TCP-only means it would always be `"tcp"`. One line to
  add back with UDP.

## Peer matching

Key sockets by the **reversed endpoint pair**: store `local + "->" + remote`,
look up `remote + "->" + local`.

```
us:    127.0.0.1:54321->127.0.0.1:8080
peer:  127.0.0.1:8080->127.0.0.1:54321   <- the key we look up
```

A TCP connection *is* its 4-tuple, so this has exactly zero or one hit.

Do **not** copy px's `_local_endpoint_to_pid` map (`px_ipc_map.py:206`), which
maps a single local endpoint to a single pid and lets later entries overwrite
earlier ones. Endpoints are shared in practice — a listener plus every forked
child that accepted on it, and dual-stack listeners. Measured on a quiet laptop:
`2 n*:7000`, `2 n*:5000`. On a multi-worker server as root, px misattributes
connections to whichever process it parsed last.

A socket inherited across a fork does put the same key under two PIDs, so the
index can collide after all. **Lowest PID wins** — arbitrary, but it has to be
something, or the same connection gets attributed to a different process every
time the page is opened, which is the nondeterminism px is faulted for above.

Known limit, which the px approach doesn't solve either: the two ends may render
the same interface differently (`127.0.0.1` vs `::ffff:127.0.0.1`).

## Direction

lsof reports state explicitly, no inference from the presence of `->`:

```
n192.168.50.32:57245->34.107.243.93:443
TST=ESTABLISHED
```

States seen on a quiet laptop: 289 `ESTABLISHED`, 10 `LISTEN`, 3 `CLOSE_WAIT`.

The rule:

> If our own local endpoint is one **we** listen on, the peer dialed us.
> Otherwise we dialed the peer.

Sound rather than heuristic — not listening on a port means we cannot have
accepted on it. It needs only our own listen set, never the peer's, so it works
identically for local and remote peers at any privilege level.

Wildcard listeners need care: a listener on `*:8080` accepts a connection whose
local endpoint reads `192.168.50.32:8080`. So match against two sets, one
port-only for `*:`-bound listeners and one of full endpoints.

`Direction` has exactly two values. There is no undeterminable case **for TCP**.

**This is why UDP is excluded.** UDP files carry no `TST` at all, and every UDP
socket is bound as soon as it sends, so there is no listen-set equivalent and
direction is genuinely undeterminable. Including UDP would mean reintroducing a
`DirectionUnknown` rendered `<->`. Deferred as a self-contained follow-up: one
enum value, one render path, one flag change.

Self-connections (a process dialing its own listening port) are **shown, once** —
deduped by keeping the socket whose local endpoint sorts first. px drops these
entirely (`px_ipc_map.py:173`), which loses information a diagnostic page should
surface. Showing both sides would report `(×2)` for one connection.

## Aggregation

**Dedup by (pid, fd) before aggregating**, as cheap insurance. px claims that on
Linux lsof reports the same open file once per thread of a process
(`px_ipc_map.py:61`, handled there by putting all files in a set). That does
**not** reproduce on lsof 4.99.4 — a process with 8 threads holding 1 connection
reported exactly 1 socket, see "Verified on Linux" above. px was presumably
looking at an older lsof. The dedup costs nothing and protects against whatever
lsof is on the target box, but it is not load-bearing.

Sockets that neither listen nor have a peer are **dropped**: lsof reports those
for a socket that is bound but was never connected, and they carry nothing worth
a line. The parser passes them through as they are, so that this decision lives
in one place.

Then aggregate by (direction, peer, port), carrying a `Count`.

Not optional. Measured on a quiet laptop, non-root: one process had **252
connections, all to the same peer endpoint**. px renders that as 252 identical
lines. A count is strictly more informative, and it means 252 connections are
one DNS lookup.

**Listening rows collapse to one and never show a count.** Dual-stack listeners
produce two identical rows (one IPv4, one IPv6 file on the same port), which
would otherwise render `tcp 7000 (listening) (×2)` — ugly and meaningless for
what is one logical listening port.

No line cap. After aggregation the pathological case is repetition, not variety,
and the output goes into a pager that handles long content. Add a cap when a
machine demands it.

## Rendering — `internal/ftop`

Two files, one per section, matching the one-file-per-section convention from
commit 6bcd409. `ipcConnectionsForPaging` grows an `allProcesses` parameter for
pid→name; `pageprocessinfo.go` already has it in hand.

```
Inter Process Communication
<Detected: TCP. Not detected: UDP, pipes, unix sockets>
curl(999) --> picked(42)                 tcp 8080
              picked(42) --> sshd(123)   tcp 22

Network Connections
            picked(42)                     tcp 8080 (listening)
1.2.3.4 --> picked(42)                     tcp 8080 (×12)
            picked(42) --> api.github.com  tcp 443 (×7)
```

- **Arrow points client → server.** "Who dialed whom" is the one directional
  fact about a TCP connection that is knowable, and it tells the reader which
  side is the service.
- **The description column is the protocol plus the server port, never an
  address.** For local IPC the address is always loopback and adds nothing; for
  a remote peer the address is already in the peer column, so printing it again
  would duplicate it on one line.
- **Partition on `Peer.Pid != 0`** — IPC section for process peers, Network
  Connections for remote hosts and listeners. A listening socket has no peer at
  all, so it lands in Network, directly above the incoming connections it
  explains.
- **Sort**: listening, then incoming, then outgoing; within a group by peer name,
  then numeric PID, then port. Grouping by line kind keeps each column block
  contiguous instead of the left column blinking in and out. One comparator for
  both sections. Name-then-numeric-PID matches the existing `CwdFriends`
  comparator. Port is the tiebreaker that makes several ports on one peer come
  out in a defined order rather than in map order.
- **The left column is only as wide as it needs to be**, so a section with
  nothing incoming starts its lines at the process instead of indenting past an
  arrow nothing uses.
- **Columns are measured per section**, not shared across the two. There is a
  title bar and two blank lines between them, so a few columns of offset is
  invisible. Sharing would force the two section functions to stop being
  independent, which is what lets each render its own error state.
- **`u.highlight()` for the current process**, not hand-rolled bold. Every other
  section does this — see `pagecommandline.go:23`, `pagelaunchhierarchy.go:37`.
  The bold in the current mockup is scaffolding.
- Build plain strings for measuring with `utf8.RuneCountInString` and a parallel
  styled string for output, per the `pagelaunchhierarchy.go:29-42` idiom.
- **The caveat line goes above the connections**, not below. A caveat that
  changes how you read a list has to arrive before the list — and this page
  streams into a pager, so a reader may never scroll to a footnote. It also
  gives the empty case a non-blank section body for free.
- The caveat line is **IPC section only**. Network Connections has nothing
  missing. Update it as kinds land (`<Detected: TCP, pipes. Not detected: ...>`)
  and delete it when nothing is missing.
- Error and empty states follow `pagecwdfriends.go`: `<Unable to list sockets:
  boom>` on outright lsof failure, `<No connections found>` under the caveat
  line when there are none.

## Reverse DNS

Only for `Peer.Pid == 0`. Dedup addresses first, resolve all of them
concurrently under **one shared** `context.WithTimeout` of 2 s — shared so the
total page cost is a constant you choose rather than scaling with peer count.
Fall back to the raw address, which is always a correct answer.

Log **once** with a count, not per address: on a big box, 200 unresolvable peers
would otherwise mean 200 near-identical log lines.

macOS note: Go uses the system (cgo) resolver for `LookupAddr`, so the deadline
returns control to the caller on time but the underlying syscall keeps running
in its goroutine. Harmless here; noted so it doesn't look like a leak in a
trace.

## Tests

- Parser: inline NUL-terminated strings, per `cwds_test.go:12-27`. This repo has
  no `testdata/` directory and shouldn't grow one for this.
- Pure connection building: unit tests with no lsof fork. Cover pair matching,
  direction including wildcard listeners, aggregation, listening rows,
  self-connections.
- Page sections: `var` seams for **both** the lsof call and the DNS resolution —
  `var getSocketsByPid = processes.GetSocketsByPid`, following
  `pagecwdfriends_test.go:15-26`. Without a DNS seam every page test does real
  network lookups.
- Page assertions are **full-block equality after ANSI stripping**, not
  `stringsContains` fragments. Alignment is the feature here, and a test
  asserting `"sshd(123)"` passes whether or not the columns line up. The
  expected string doubles as documentation of what the section looks like.
  Keep `stringsContains` for error and empty paths, where there is no layout.
- `stripAnsi` helper: `twin` has none, and moor's `StripFormatting` is under
  `internal/` so it can't be imported. Write a small one — everything `twin`
  emits is an SGR sequence, so `\x1b\[[0-9;]*m` covers it. Comment that it only
  handles SGR and should be improved if more is ever needed.
- `./test.sh` before any PR, per `AGENTS.md`.

## Deferred, deliberately

- Pipes (`PIPE`/`FIFO`) — the display grammar in the original mockup
  (`grep(1234) | proc | sort(1234)`) is designed for these. Note that combining
  a stdin peer and a stdout peer onto one line has no TCP equivalent, so that
  rendering question is still open.
- Unix domain sockets.
- UDP — needs `DirectionUnknown` and a `<->` render path.
- `(ssh)` service-name annotations next to port numbers. Cosmetic, needs
  `/etc/services` parsing, no model change.
- A line cap for processes with hundreds of *distinct* peers.
- Sharing one lsof invocation across sections, once there are three of them.
