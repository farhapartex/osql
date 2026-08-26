<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Summary

`summary` gives you a folder at a glance: how much is in it, how big, what kind
of thing, and what is taking the space.

```bash
summary from 'Downloads'
```

```
Downloads — one level

  WHAT       COUNT       SIZE
  files         52     1.4 GB
  folders        8          —
  total         60     1.4 GB

  TYPE       COUNT       SIZE
  pdf           18   890.2 MB
  zip            6   412.1 MB
  txt           21     1.2 MB
  and 10 more types

  LARGEST                                              SIZE
  report.pdf                                       412.0 MB
  Designing Data Intensiv…y Martin Kleppmann.pdf    23.8 MB

  MODIFIED  2024-03-11 to 2026-08-25
```

## One level or all levels

By default `summary` looks at the folder itself and nothing deeper. The heading
says which, because it changes what the numbers mean:

```bash
summary from 'Downloads'              # Downloads — one level
summary from 'Downloads' recursive    # Downloads — all levels
```

**At one level, the size only covers files sitting directly in that folder.** If
you want "how big is this folder, really", you want `recursive`.

## Reading it

**WHAT** — how many files and folders. Folders never carry a size, so the size
column is the same on the `files` and `total` rows.

**TYPE** — the biggest file types, largest first. Files with no extension group
under `—`. Only the top 5 are shown; if there are more you will see a line like
`and 12 more types`.

**LARGEST** — the biggest files, largest first. Top 5 again. Long names are
shortened in the middle with a `…`, keeping the start and the extension, so one
long filename cannot push the SIZE column off the side of your screen.

**MODIFIED** — the oldest and newest change dates in the folder.

## Installed apps

`summary apps` gives the same treatment to the apps on your machine — counts,
sizes grouped by where they came from, and the biggest ones. See
[Installed apps](apps.md).

```bash
summary apps
```

## Folders that get skipped

Folders like `node_modules`, `venv`, `.venv`, `__pycache__` and `.git` hold
thousands of files you almost never mean to include. `summary` walks past them
and tells you it did:

```bash
summary from 'my-project' recursive
```

```
Skipped 2 folders: node_modules, .venv
Add "with skipped" to include them — it will take longer.
```

To count them anyway, add `with skipped`:

```bash
summary from 'my-project' recursive with skipped
```

Expect it to be slower. That is the whole reason they are skipped by default.

## Short answers

An empty folder gets one line:

```bash
summary from 'new-folder'
```

```
'new-folder' is empty.
```

A folder holding only other folders gets a short line too, since there is
nothing to measure:

```bash
summary from 'src'
```

```
src — one level

  Contains 3 folders, and no files.
```

## Why it can be slow

Unlike listing, `summary` has to read the size and date of every file it counts.
Listing can skip that work; a summary cannot, because those numbers *are* the
answer.

Memory stays small no matter how big the folder is — `osql` keeps only the top
few files and a tally per type, never the whole list. But on a very large tree,
`summary … recursive` will take a moment.

## Not supported yet

`summary` takes no `where` clause. It describes a whole folder, not a filtered
part of one. Use [counting](counting.md) if you want a filtered number:

```bash
count(files) from 'Downloads' where type = 'pdf'
```

## Next

- [Counting](counting.md) — a single number, with filters
- [Queries](queries.md) — listing what is actually there
