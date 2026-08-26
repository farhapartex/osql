<p align="center">
  <img src="osql_chevron.png" alt="osql" width="380">
</p>

Ask your filesystem questions in plain sentences instead of remembering flags.

`osql` opens a small shell where you type things like `files from 'Documents'`
and get a clean table back. No `find`, no pipes, no man pages. When something
goes wrong, the message is written to be read by a person.

It has no third-party dependencies — everything is Go's standard library.

## Quick start

Build it:

```bash
make build
```

Run it:

```bash
./bin/osql
```

Then try a few things:

```bash
files from 'Documents'
folders from 'Documents' where count(child) > 5
count(all) from 'Documents'
summary from 'Documents' recursive
open 'Documents/notes.txt'
new file 'Documents/notes.txt' data='hello'
```

Press `Ctrl+D` or type `exit` to leave.

## Documentation

| Page | What it covers |
|---|---|
| [Installation](docs/installation.md) | Building, installing, and what you need first |
| [Queries](docs/queries.md) | Listing files and folders, and how paths work |
| [Filtering](docs/filtering.md) | The `where` clause: names, types, and patterns |
| [Counting](docs/counting.md) | Getting a number instead of a list |
| [Summary](docs/summary.md) | A folder at a glance: sizes, types, biggest files |
| [Opening files](docs/opening.md) | Printing what is inside a text file |
| [Creating](docs/creating.md) | Making new files and folders |
| [Deleting](docs/deleting.md) | Removing files and folders, safely |
| [Output](docs/output.md) | Reading the table and the size column |
| [Shell and flags](docs/shell.md) | Built-in commands and command-line options |
| [Error messages](docs/errors.md) | What each message means and how to fix it |
| [Your files](docs/files.md) | What osql saves on your machine |
| [Development](docs/development.md) | Building, testing, and the project layout |

## Status

Early days, and honest about it. Listing, filtering, counting, summarising,
reading, creating, and deleting all work, and the prompt has arrow-key editing
with history. See [known limits](docs/shell.md#known-limits) for what is missing.

macOS and Linux only.
