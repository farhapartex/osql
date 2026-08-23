# osql

An interactive shell for querying your filesystem in SQL-like statements instead
of flags and pipes.

```
$ osql
osql > select files from 'Documents'
osql > select files from 'Documents' where type = 'txt'
osql > select files from '~' recursive where name_like = '%report%'
osql > exit
```

Results come back as a table of name, type, size, and modified time. Errors are
written to be read by a person, not parsed from a stack trace.

`osql` has no third-party dependencies — the shell, lexer, parser, and executor
are all standard library.

## Status

Early development. The build and package skeleton are in place; the query engine
is not implemented yet, so the binary currently reports its version and exits.

## Requirements

- Go 1.26 or newer
- macOS or Linux

## Build

```
make build
```

The binary lands in `bin/osql`.

## Run

```
./bin/osql
```

## Install

Puts `osql` on your `PATH` via `$GOBIN` (or `$GOPATH/bin`):

```
make install
osql
```

## Other targets

| Target | Does |
|---|---|
| `make test` | `go test ./... -race` |
| `make bench` | benchmarks with allocation counts |
| `make vet` | `go vet ./...` |
| `make fmt` | `go fmt ./...` |
| `make clean` | removes `bin/` |
| `make all` | vet, test, then build |

## Planned query syntax

```
select <all|files|folders> from '<path>' [recursive] [where <condition>]
```

Fields are `name`, `name_like` (with `%` wildcards), `type`, and `count(child)`.
Conditions combine with `and`. Queries read one directory level unless you add
`recursive`.
