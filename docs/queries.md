<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Queries

Every query starts with **what you want**, then **where to look**.

```bash
files from 'Documents'
```

There is no verb like `select` in front. If you type one, `osql` will tell you to
drop it.

---

**On this page**

- [Four things you can ask for](#four-things-you-can-ask-for)
- [How paths work](#how-paths-work)
- [Moving around](#moving-around)
- [Looking inside subfolders](#looking-inside-subfolders)
- [Folders that are skipped](#folders-that-are-skipped)
- [Small conveniences](#small-conveniences)
- [Special characters in quotes](#special-characters-in-quotes)

## Four things you can ask for

```bash
files from 'Documents'      # only files
folders from 'Documents'    # only folders
all from 'Documents'        # both
```

```bash
apps                        # installed apps, not files
```

Always use the plural. `file` and `folder` are not accepted — `osql` will
correct you.

`apps` is the odd one out: it takes no folder, because apps live wherever your
system put them. See [Installed apps](apps.md).

## How paths work

`osql` starts in the folder you ran it from, and paths work the way they do in
your terminal. Nothing special to learn.

```bash
files from 'Documents'        # a folder next to you
files from './Documents'      # the same thing
files from 'Documents/2026'   # nested
files from '..'               # the folder above you
files from '/var/log'         # an exact path from the top of the disk
files from '~/Downloads'      # under your home folder
files from '.'                # the folder you are in
```

There is nothing osql will not look at. If you can read it, osql can list it.
If you cannot, it says so:

```bash
files from 'nowhere'
```

```
I couldn't find a folder at 'nowhere'. Check the path and try again.
```

## Moving around

Use `cd`, and `pwd` when you lose track. The prompt shows where you are.

```bash
cd Documents          # go in
cd ..                 # go up
cd /var/log           # jump anywhere
cd ~                  # go home
cd                    # also go home
cd -                  # back where you just were
pwd                   # print the full path
```

```
osql ~/Documents/goupp > cd internal
osql ~/Documents/goupp/internal >
```

If you want to start somewhere other than where you are, say so when you launch:

```bash
osql --dir /var/log
```

## Looking inside subfolders

By default a query looks at one level only, like `ls`. Add `recursive` to go all
the way down:

```bash
files from 'Documents'              # just the top level
files from 'Documents' recursive    # every subfolder too
```

Recursive results show the path relative to where you started:

```
NAME                     TYPE  SIZE     MODIFIED
notes.txt                txt   4.2 KB   2026-08-20 14:02
2026/q4-report.xlsx      xlsx  88.4 KB  2026-08-01 11:14
```

Recursive is opt-in on purpose. `all from '/'` should print a screenful, not
scan your whole home folder.

## Folders that are skipped

When going recursive, `osql` walks past folders nobody wants in results:

`.git`, `node_modules`, `venv`, `.venv`, `__pycache__`, `.Trash`,
`.Spotlight-V100`, `.fseventsd`, `Library/Caches`, `Library/Containers`

This usually removes most of the work and makes recursive searches feel fast.
Hidden files like `.gitignore` **are** shown — only these folders are skipped.

[`delete`](deleting.md) is the one command that does *not* skip them, so it can
never tell you a folder is empty while files remain inside.

## Small conveniences

Keywords ignore capitals:

```bash
FILES FROM 'Documents'
```

A trailing semicolon is fine, if that is your habit:

```bash
files from 'Documents';
```

Quotes are optional when the path has no spaces:

```bash
files from Documents
```

File and folder names keep their capitals, though. `'.TXT'` and `'.txt'` are
different things.

## Special characters in quotes

Inside quotes, a backslash starts an escape:

| You type | You get |
|---|---|
| `\n` | a new line |
| `\t` | a tab |
| `\r` | a carriage return |
| `\\` | one backslash |
| `\'` | an apostrophe |

So a folder with an apostrophe in its name is written like this:

```bash
files from 'Ali\'s Documents'
```

Anything else after a backslash is an error, which stops a typo from quietly
becoming part of a name.

<!-- nav -->

---

[← Installation](installation.md) · [All pages](README.md) · [Filtering →](filtering.md)
