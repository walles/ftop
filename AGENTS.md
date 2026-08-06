# Validation

Use `./test.sh` to run all tests. In addition to just running all tests, that
script will do linting, some cross compiling and more.

# Running The Tests On Linux

**Not part of the routine.** `./test.sh` on a Mac is what a change has to pass,
and CI on `main` covers Linux — most changes can't break it specifically, so
don't reach for a container by default.

It earns its few minutes when a change reads output that differs per platform,
`lsof` and `ps` above all, or when a Linux-only path is the thing being changed.
`./test.sh` on a Mac builds the Linux binaries but runs none of them, so those
are the cases where passing locally says the least. A container runs the real
thing:

```bash
docker run --rm -it -v "$PWD:/src" -v ftop-gomod:/go/pkg/mod \
  -v ftop-gobuild:/root/.cache/go-build -w /src golang:1.25 bash
```

Then, inside:

```bash
apt-get update && apt-get install -y lsof procps wtmpdb
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
  | sh -s -- -b "$(go env GOPATH)"/bin v2.8.0
wtmpdb boot
./test.sh
```

Every line of that is load bearing, so adapt rather than drop: the image has
neither `lsof` nor the `ps` ftop forks, `test.sh` insists on the golangci-lint
version `.github/workflows/linux-ci.yml` installs, and `wtmpdb boot` gives `last`
a database, without which `TestGetUsersAtSmoke` fails. On Debian 13 `last` comes
from `wtmpdb` and not from `util-linux`.

Run `test.sh` from the `docker run` shell rather than through a later
`docker exec`: an exec'd process has no parent inside the container's PID
namespace, so `TestGetAll`'s walk up to init lands on the shell instead of on PID
1. The two cache volumes are what makes a second run cost seconds. Add
`--privileged` if the test needs to mount anything.

Driving the terminal UI in there needs a pty with a real window size, which
`docker exec` does not give it: wrap it as
`script -qec "stty rows 60 cols 170; ./ftop" /dev/null` and feed keystrokes on
stdin with `sleep`s between them.

# Manual Testing

If the user wants to test manually, ask them to run `./ftop.sh` rather than
building a binary yourself. `ftop.sh` builds and runs the current sources with
race detection enabled.

# Test Conventions

Tests that parse a subprocess' output use inline fixture strings, per
`cwds_test.go`. This repo has no `testdata/` directory and shouldn't grow one.

Assert a page section against a whole expected block, via `sectionBody()`, rather
than picking fragments out of it. Alignment is a feature of those sections.

# Fixing Bugs

1. Create a new branch with a sensible name, e.g. `fix-crash-on-search-backwards`.
2. Reproduce the bug with a failing test. Commit this test once it reproduces
   the bug properly.
3. Fix the bug until your new test passes.

# PR Best Practices

Always run `./test.sh` locally before making any PRs.

# Releases

Release messages go into annotated tags. Please look at the most recent
annotated tags for style guidance. The basis for all those messages are user
visible changes since last release.
