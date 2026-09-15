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
- [How big is a folder](#how-big-is-a-folder)
- [Putting results in order](#putting-results-in-order)
- [Asking for only the first few](#asking-for-only-the-first-few)
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

## How big is a folder

Folders show `—` in the SIZE column, because a folder has no size of its own —
it is whatever is inside it. Add `with size` and osql adds it up:

```bash
folders from '~' with size
```

```
NAME         TYPE    SIZE      MODIFIED
Documents    folder  4.2 GB    2026-03-01 09:12
Downloads    folder  18.7 GB   2026-03-02 14:21
Pictures     folder  61.3 GB   2026-02-11 08:40

3 files
```

It goes after the path and before `where`, the same place it goes for
[apps](apps.md).

This is the one query that asks osql to do real work: adding up a folder means
walking everything inside it. That is why it is something you ask for rather
than something you always get — a plain `folders from '~'` stays instant.

Put together with sorting, this answers the question people actually have:

```bash
folders from '~' with size sorted by size desc limit 10
```

That is "the ten biggest folders in my home directory".

Sorting folders by size needs `with size`, because until they are measured there
is nothing to sort. osql says so if you forget.

A folder osql cannot fully read keeps its `—` rather than reporting a total it
does not know, and it is left out of the count.

## Putting results in order

`sorted by` orders the results. Ascending is the default; add `desc` to reverse
it:

```bash
files from 'Downloads' sorted by size desc
files from 'Documents' sorted by modified desc
files from 'src' sorted by name
```

You can sort by `name`, `type`, `size` and `modified`. `size` is for files, so
sorting folders by it does not work yet. Files with the same value are ordered
by name, so the same query always gives the same output.

It goes after `where` and before `limit`. The two together answer the question
people actually ask:

```bash
files from '~' recursive sorted by size desc limit 20
```

That is "the twenty biggest files anywhere under my home folder".

### Sorting looks at everything

Without a sort, `limit 20` stops the search at the twentieth match. **With a
sort it cannot**, because the biggest file might be the last one found. So a
sorted query always searches the whole folder, and the limit decides how many of
the results you keep rather than when to stop looking.

That costs time on a big folder, but the answer is exact: `limit 20` with a sort
really is the top twenty, not the first twenty that turned up.

If you sort **without** a limit, osql holds up to 10,000 results. Past that it
says so:

```
Sorted the first 10000 matches. Add a limit to be sure you are seeing the top.
```

The rows you get are still genuinely the top ones — only the tail is missing.

## Asking for only the first few

`limit` stops after a set number of results:

```bash
files from 'Downloads' limit 10
files from '~' recursive where size > 100mb limit 20
```

It goes at the end, after `recursive` and after `where`.

This is not just tidier output — **osql stops looking as soon as it has
enough.** On a large folder `limit 10` finishes almost immediately, because the
search ends at the tenth match instead of reading everything and then throwing
most of it away.

When the limit is what stopped the search, osql says so, since there may be
more:

```
10 files
Showing the first 10. Raise the limit to see more.
```

If fewer results than the limit come back, that line does not appear, so you
know you are seeing everything.

A limit needs to be 1 or more, and it does not go with `count(...)` — a count is
already a single number, so limiting it would change nothing.

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
