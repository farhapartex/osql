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

`osql` starts at your home folder and never looks outside it. So all three of
these mean the same folder:

```bash
files from 'Documents'
files from '/Documents'
files from '~/Documents'
```

A leading `/` means *your home folder*, not the top of the disk. This is the one
surprise worth remembering.

These all mean your home folder itself:

```bash
files from '.'
files from '~'
files from '/'
```

Nested paths work as you would expect:

```bash
files from 'Documents/2026/reports'
```

### Going outside home

You cannot, by design:

```bash
files from '../etc'
```

```
I can only look inside '/Users/you'. '../etc' points outside it.
```

If you need somewhere else, start `osql` with a different starting point:

```bash
osql --root /
osql --root /var/log
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
