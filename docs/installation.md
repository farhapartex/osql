<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Installation

## What you need

- Go 1.26 or newer
- macOS or Linux

That is all. `osql` uses no outside libraries, so there is nothing to download.

## Build it

```bash
make build
```

The program lands in `bin/osql`. Run it from there:

```bash
./bin/osql
```

## Put it on your PATH

If you want to type `osql` from anywhere:

```bash
make install
```

This puts it in your `$GOBIN` folder (or `$GOPATH/bin` if `GOBIN` is not set).
Then just:

```bash
osql
```

If the command is not found, your Go bin folder is probably not on your PATH.
Add it to your shell config:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Check it works

```bash
osql --version
```

You should see something like `osql v0.1.0 (a1b2c3d)`.

## First run

The first time you start `osql`, it makes a small folder at `~/.osql` to keep
your command history. Nothing leaves your machine. See
[Your files](files.md) for what is in there.

## Other build commands

```bash
make test     # run the Go tests
make e2e      # run the real program against a test folder
make vet      # check for common mistakes
make clean    # delete bin/
```
