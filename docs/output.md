<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Output

Every listing comes back as the same four columns.

```
NAME            TYPE    SIZE     MODIFIED
notes.txt       txt     4.2 KB   2026-08-20 14:02
report.pdf      pdf     1.1 MB   2026-08-19 09:31
archive.tar.gz  gz      2.3 GB   2026-08-18 22:07
Makefile        —       570 B    2026-08-24 00:20
goupp           folder  —        2026-08-22 17:23

5 files
```

## The columns

**NAME** — the file or folder name. For a `recursive` query it is the path
relative to where you started, like `2026/q4-report.xlsx`.

**TYPE** — the extension without its dot (`txt`, `pdf`, `gz`), or `folder` for a
folder, or `—` when a file has no extension. Whatever you see here works in a
query:

```bash
files from 'Documents' where type = 'gz'
```

**SIZE** — how big the file is, in a unit that keeps the number readable.
Folders show `—`.

**MODIFIED** — when it last changed, as `YYYY-MM-DD HH:MM`.

## The em dash

`—` means "nothing to show here". You will see it for a file with no extension,
and for the size of any folder.

## Why folders have no size

Adding up a folder means walking everything inside it. Doing that for every row
of `all from '/'` would read your whole home folder just to draw one screen. So
folders show `—`, and you ask for totals when you actually want them.

## Sizes

Sizes use the largest unit that keeps the number at 1 or above:

| Bytes | Shown as |
|---|---|
| 938 | `938 B` |
| 1024 | `1.0 KB` |
| 123456 | `120.6 KB` |
| 1153434 | `1.1 MB` |
| 2469606195 | `2.3 GB` |

Bytes get no decimal point. Everything above bytes gets one. Steps are 1024, the
same as `ls -h`.

You will never see `1024.0 KB` — at that point it becomes `1.0 MB`.

## The footer

The line at the bottom counts the rows:

```
5 files
```

It always says `files`, even for one result or for folders. It is a row count,
not a description.

## When nothing matches

You get a plain sentence instead of an empty table:

```
No files matched.
```

And if the folder itself is empty:

```
'Documents' is empty.
```

Neither one is an error.

## Next

- [Counting](counting.md) — when you want the number, not the rows
