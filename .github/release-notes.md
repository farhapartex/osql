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

## New in this release

**Find what is using your disk.** Filter by `size` and by `modified`, put results in order with `sorted by`, and stop early with `limit`:

```bash
files from '~' recursive where size > 100mb sorted by size desc limit 20
files from 'Downloads' where modified > '7 days ago'
```

**Folders can be measured.** A folder has no size of its own, so `with size` adds up what is inside it. Together with sorting, that answers the obvious question:

```bash
folders from '~' with size sorted by size desc limit 10
```

**Long searches behave properly.** A search over a big folder shows a running count while it works, and **Ctrl+C now stops the query and gives you the prompt back** instead of closing osql.

`limit` on its own ends the search as soon as it has enough, so it is quick as well as tidy. Add a sort and osql looks at everything before answering, because the biggest file might be the last one found — so `sorted by size desc limit 20` really is the top twenty.

## What works

Listing and filtering files and folders, counting, folder summaries, installed apps, reading text files, creating files and folders, deleting to the trash with a preview and a typed confirmation, moving around with `cd`, and arrow-key line editing with history.

Changed your mind? `osql uninstall` removes the program and `~/.osql`, after showing you exactly what it will delete. It never asks for `sudo`, and `--keep-data` keeps your command history.

Queries read like sentences, so there is no flag order to get wrong:

```bash
files from 'Documents' where type = 'pdf'
files from '~' recursive where size > 100mb sorted by size desc limit 20
folders from '~' with size sorted by size desc limit 10
files from 'Downloads' where modified > '7 days ago'
summary from 'Downloads' recursive
apps with size
count(files) from 'src' recursive
```

macOS and Linux, on Intel or ARM. No dependencies — Go's standard library, start to finish, which is why the binary is about 2.5 MB and needs no runtime.

## Getting started

[The documentation](https://github.com/farhapartex/osql/blob/main/docs/README.md) starts with installation and your first query. [Error messages](https://github.com/farhapartex/osql/blob/main/docs/errors.md) lists every message osql can print and how to fix it, and [known gaps](https://github.com/farhapartex/osql/blob/main/docs/errors.md#known-gap) covers what is still missing.

Found something broken? [Open an issue](https://github.com/farhapartex/osql/issues) — early reports are the most useful ones.
