Use `test.sh` to run all tests. In addition to just running all
tests, it will do linting, some cross compiling and more.

Log levels are debug, info and error, as described in internal/log/log.go:

- debug: Will be shown to the user with --debug flag only
- info: Will be included in the information printed after any crash
- error: Always shown to the user after exit, should be reported as bugs

In a lot of ways this is a Go rewrite of <https://github.com/walles/px>.
The `ftop` binary corresponds to `px`'s `ptop` binary. If px is cloned
right next to ftop, feel free to look at the px source code for
inspiration!

The `twin` TUI toolkit may also be checked out under `../moor/`. Feel
free to poke around there for details.

Release notes are written as git annotated tag messages. The first line
should be a short release title, the second line should be blank, and
subsequent lines are the release description. Keep them terse and
hand-written in tone, and avoid commit-dump style release notes unless
explicitly asked.
