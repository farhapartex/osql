<p align="center">
  <img src="osql_chevron.png" alt="osql" width="380">
</p>

<p align="center">
  <b>Ask your filesystem questions in plain sentences.</b><br>
  No flags to remember, no pipes to build, no man pages to read.
</p>

<p align="center">
  <a href="#quick-start">Quick start</a> ·
  <a href="docs/README.md">Documentation</a> ·
  <a href="docs/errors.md">Error messages</a>
</p>

---

`osql` opens a small shell where you describe what you are looking for and get a
clean table back.

```
osql > files from 'Documents' where type = 'pdf'

NAME                  TYPE  SIZE      MODIFIED
invoice-april.pdf     pdf   248.1 KB  2026-03-02 14:21
lease-agreement.pdf   pdf   1.2 MB    2026-01-18 09:03

2 files
```

Three things shape the whole tool:

- **Questions read like sentences.** `files from 'Downloads' where name_like =
  '%report%'` is the whole query. There is no verb to learn, and no flag order to
  get wrong.
- **Mistakes are explained, not just reported.** Every message says what went
  wrong *and* what to type instead. Ask for `file` and it tells you that you
  wanted `files`.
- **Nothing to install but the program.** No third-party libraries — Go's
  standard library, start to finish.

## Quick start

You need **Go 1.26 or newer**, on **macOS or Linux**.

```bash
make build      # builds ./bin/osql
./bin/osql
```

Then try a few things:

```bash
files from 'Documents'                     # what is in here?
cd Documents                               # move around, like a normal shell
folders from 'Documents' where count(child) > 5
count(all) from 'Documents'                # just the number
summary from 'Documents' recursive         # sizes and biggest files
apps                                       # what is installed
open 'Documents/notes.txt'                 # print a text file
```

Type `help` to see everything, and `exit` or `Ctrl+D` to leave.

To use `osql` from any folder, run `make install` — see
[Installation](docs/installation.md).

## Documentation

Start here:

| Page | What it covers |
| --- | --- |
| [Installation](docs/installation.md) | What you need, building, and your PATH |
| [Queries](docs/queries.md) | Your first query, and how paths work |
| [Filtering](docs/filtering.md) | The `where` clause: names, types, and wildcards |
| [Error messages](docs/errors.md) | What a message means and how to fix it |

**[See all 14 pages →](docs/README.md)**
