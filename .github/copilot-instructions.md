Use `test.sh` to run all tests. In addition to just running all
tests, it will do linting, some cross compiling and more.

Log levels are debug, info and error, as described in internal/log/log.go:

- debug: Will be shown to the user with --debug flag only
- info: Will be included in the information printed after any crash
- error: Always shown to the user after exit, should be reported as bugs
