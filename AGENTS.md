# Validation

Use `./test.sh` to run all tests. In addition to just running all tests, that
script will do linting, some cross compiling and more.

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
