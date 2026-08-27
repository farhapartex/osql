osql is an early release. It works, and it is honest about what it cannot do yet — expect rough edges, and expect breaking changes between 0.x versions.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/farhapartex/osql/main/install.sh | sh
```

That picks the right build for your machine, checks it against the SHA256 published below, and installs to `~/.local/bin` without asking for `sudo`.

Or with Go:

```bash
go install github.com/farhapartex/osql@__VERSION__
```

Or download a tarball below, unpack it, and move `osql` onto your `PATH`. Every archive carries its own `LICENSE` and `README.md`, and `checksums.txt` lets you verify what you downloaded:

```bash
sha256sum -c checksums.txt      # shasum -a 256 -c on macOS
```

## What works

Listing and filtering files and folders, counting, folder summaries, installed apps, reading text files, creating files and folders, deleting to the trash with a preview and a typed confirmation, moving around with `cd`, and arrow-key line editing with history.

Queries read like sentences, so there is no flag order to get wrong:

```bash
files from 'Documents' where type = 'pdf'
folders from '.' where count(child) > 5
summary from 'Downloads' recursive
apps with size
count(files) from 'src' recursive
```

macOS and Linux, on Intel or ARM. No dependencies — Go's standard library, start to finish, which is why the binary is about 2.5 MB and needs no runtime.

## Getting started

[The documentation](https://github.com/farhapartex/osql/blob/main/docs/README.md) starts with installation and your first query. [Error messages](https://github.com/farhapartex/osql/blob/main/docs/errors.md) lists every message osql can print and how to fix it, and [known gaps](https://github.com/farhapartex/osql/blob/main/docs/errors.md#known-gap) covers what is still missing.

Found something broken? [Open an issue](https://github.com/farhapartex/osql/issues) — early reports are the most useful ones.
