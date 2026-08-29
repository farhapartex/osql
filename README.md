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

Install it:

```bash
curl -fsSL https://raw.githubusercontent.com/farhapartex/osql/main/install.sh | sh
```

That picks the right build for your machine, verifies its checksum, and puts
`osql` in `~/.local/bin` without asking for `sudo`. Go users can
`go install github.com/farhapartex/osql@latest` instead, and
[Installation](docs/installation.md) covers building from source.

Then run it:

```bash
osql
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

Other ways to install — Go, a tarball, or from source — are in
[Installation](docs/installation.md).

To remove it again:

```bash
osql uninstall
```

That shows you exactly what it will delete and waits for you to type `yes`. Add
`--keep-data` to keep your command history at `~/.osql`.

## Documentation

Start here:

| Page | What it covers |
| --- | --- |
| [Installation](docs/installation.md) | What you need, building, your PATH, and removing it |
| [Queries](docs/queries.md) | Your first query, and how paths work |
| [Filtering](docs/filtering.md) | The `where` clause: names, types, and wildcards |
| [Error messages](docs/errors.md) | What a message means and how to fix it |

**[See all 14 pages →](docs/README.md)**

## License

MIT — see [LICENSE](LICENSE). Use it, change it, ship it; just keep the
copyright notice.
