<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Development

How to build it, how to test it, and how to find your way around
the code.

---

**On this page**

- [Commands](#commands)
- [Layout](#layout)
- [House rules](#house-rules)
- [Two rules that are easy to break](#two-rules-that-are-easy-to-break)
- [Before you call it done](#before-you-call-it-done)
- [Releasing](#releasing)
- [License](#license)

## Commands

Clone it and build. You need **Go 1.26 or newer** and nothing else — `osql` has
no outside libraries, so there is nothing to download.

```bash
git clone https://github.com/farhapartex/osql.git
cd osql
make build
./bin/osql
```

```bash
make build       # build bin/osql
make install     # put osql on your PATH via $GOBIN
make test        # go test ./... -race
make e2e         # run the real binary against a test folder
make bench       # benchmarks with allocation counts
make vet         # go vet ./...
make fmt         # go fmt ./...
make fmt-check   # fail if anything needs gofmt
make cross       # check every release platform still builds
make dist        # build release tarballs into dist/
make version       # print what a release would be tagged
make version-check # fail if VERSION is malformed
make release-check # also fail if that tag already exists
make clean       # remove bin/ and dist/
make all         # fmt-check, vet, test, build, e2e
```

A binary built from a working copy reports the last tag plus how far past it you
are, so `osql --version` looks like `osql v0.1.0-3-gabc1234 (abc1234)` rather
than a clean release number. That is deliberate — it tells you the build is not
a release.

Run `make e2e` after any change. It builds the program, points it at a throwaway
folder, and prints a named pass or fail for every behaviour:

```
paths
  ✔ a bare folder name is relative
  ✔ dot is the current folder
  ✘ tilde form
    query: files from '~/docs'
      | expected: notes.txt
      |   actual: No files matched.

summary
  241 passed, 1 failed
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

## Releasing

The version lives in one place: the `VERSION` file at the top of the repository,
holding a bare semver number with no `v`.

```bash
cat VERSION      # 0.1.0
make version     # v0.1.0
```

To cut a release:

```bash
echo 0.2.0 > VERSION           # 1. bump it
git commit -am "release 0.2.0" # 2. commit and get it onto main
git tag v0.2.0                 # 3. tag it — v plus the file's contents
git push origin main --tags
```

Pushing the tag starts `release.yml`: it runs the whole suite on macOS and Linux
first, then builds four platforms, checks each binary reports the right version,
and publishes the release with checksums.

**The release refuses to publish if the tag and the `VERSION` file disagree**, so
a forgotten bump fails loudly instead of shipping a mislabelled binary.

**Releases only happen on a tag.** Ordinary pushes and pull requests run the
tests; they never build or publish a release.

Version numbers follow semver, and `0.x` already means "expect breaking
changes" — so no `-beta` suffixes, and GitHub's prerelease box stays unticked.
Both would quietly break installs: the suffix stops `go install @latest` from
finding the release, and the box stops `install.sh` from finding it.

## License

osql is MIT licensed. The full text is in [LICENSE](../LICENSE) at the top of the
repository.

Contributions are taken under the same license. There is no CLA to sign.

<!-- nav -->

---

[← Your files](files.md) · [All pages](README.md)
