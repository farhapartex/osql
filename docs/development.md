<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Development

How to build it, how to test it, and how to find your way around
the code.

## Commands

```bash
make build     # build bin/osql
make install   # put osql on your PATH
make test      # go test ./... -race
make e2e       # run the real binary against a test folder
make bench     # benchmarks with allocation counts
make vet       # go vet ./...
make fmt       # go fmt ./...
make clean     # remove bin/
make all       # vet, test, build, e2e
```

Run `make e2e` after any change. It builds the program, points it at a throwaway
folder, and prints a named pass or fail for every behaviour:

```
select — paths are root-relative
  ✔ bare path
  ✔ leading slash means the root, not the filesystem
  ✘ tilde form
    query: files from '~/docs'
      | expected: notes.txt
      |   actual: No files matched.

summary
  95 passed, 1 failed
```

It sets `HOME` to a temp folder, so your real `~/.osql` is never touched.

## Layout

```
main.go              wiring only — builds the parts and hands off
internal/shell/      the read-type-print loop and built-in commands
internal/reader/     reading a line from the terminal
internal/query/      turning text into a query (lexer, parser)
internal/engine/     running a query against the filesystem
internal/vfs/        the filesystem, behind an interface
internal/output/     drawing the table
internal/oerr/       every message the user sees
internal/state/      the ~/.osql folder
internal/cli/        command-line flags
internal/buildinfo/  version string
test/                all tests
scripts/e2e.sh       the end-to-end check
```

Text flows one way:

```
line → lexer → parser → executor → renderer → screen
```

## House rules

**No outside libraries.** Standard library only. `go.mod` has no `require`
block, and that is deliberate — it is why the release build needs no C compiler
and why cross-compiling is a plain build matrix.

**No comments in the code.** Names and small functions carry the meaning. The
only `//` lines allowed are build directives like `//go:build darwin || linux`.
Reasons live in the design notes, not in source files.

**All tests live in `test/`.** Not next to the code. This means only exported
functions are reachable from a test, so anything worth testing gets exported
from an `internal/` package.

**Errors are the product.** Every message a user can see lives in
`internal/oerr` and nowhere else. Tests assert the exact strings, so changing
wording is a deliberate act.

**`fstest.MapFS` for filesystem tests**, not temp folders — except when you are
testing something only a real filesystem does, like permissions.

## Two rules that are easy to break

**Never read a file's details just to filter it.** A directory listing already
tells you the name and whether it is a folder. Size and date each cost a system
call, so only ask for them on rows that already matched. There is a test that
counts those calls and fails if you slip.

**Compile a pattern once per query, never once per file.** `name_like` patterns
are turned into a matcher before the walk starts. A test asserts the match path
allocates nothing at all.

## Before you call it done

```bash
go build ./...
go vet ./...
go test ./... -race
make e2e
```

Report what actually happened, including the failures.

<!-- nav -->

---

[← Your files](files.md) · [All pages](README.md)
