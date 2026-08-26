<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Deleting

`delete` always shows you what it is about to remove and waits for you to type
`yes`. Nothing goes without a confirmation.

```bash
delete file 'notes.txt'
```

```
This file will be deleted:

  notes.txt                                        1.2 KB

It will be moved to the trash.
Type "yes" to go ahead, anything else to cancel.
delete>
```

Type anything other than `yes` — or press Ctrl+D — and nothing happens.

---

**On this page**

- [Deleting one thing](#deleting-one-thing)
- [Deleting many things](#deleting-many-things)
- [Things go to the trash](#things-go-to-the-trash)
- [Folders show their weight](#folders-show-their-weight)
- [Nothing here is skipped](#nothing-here-is-skipped)
- [When some files will not budge](#when-some-files-will-not-budge)
- [Guards](#guards)

## Deleting one thing

Mirrors [creating](creating.md), so the pair reads the same way:

```bash
new file 'notes.txt'        delete file 'notes.txt'
new folder 'reports'        delete folder 'reports'
```

Point `delete file` at a folder and it tells you the command you wanted:

```bash
delete file 'reports'
```

```
'reports' is a folder, not a file. Try: delete folder 'reports'
```

## Deleting many things

This is the [listing](queries.md) query with `delete` in front:

```bash
delete files from 'Downloads' where type = 'tmp'
delete files from 'Downloads' where name_like = '%.log'
delete folders from 'src' where count(child) = 0
delete all from 'temp'
delete files from 'build' recursive where type = 'o'
```

**Because the filter is the same, you can always look before you leap.** Drop the
word `delete` and you see exactly the set that would go:

```bash
files from 'Downloads' where type = 'tmp'          # look
delete files from 'Downloads' where type = 'tmp'   # remove that same set
```

That habit is worth keeping.

## Things go to the trash

By default `delete` moves things to your trash, so a mistake is recoverable —
open the Trash and put it back.

If a file with the same name is already in there, `osql` keeps both by adding a
number, the way Finder does. It never overwrites something you deleted earlier.

### Deleting for good

Add `permanently` when you really mean it:

```bash
delete file 'secret.txt' permanently
```

```
It will be deleted for good, with no way back.
Type "yes" to go ahead, anything else to cancel.
```

The confirmation says so plainly, because there is no undo.

Some files cannot go to the trash — anything on a different disk, for instance.
`osql` will tell you and point you at `permanently` rather than quietly deleting
it for good.

## Folders show their weight

Deleting a folder takes everything inside it, so the preview says how much:

```bash
delete folder 'oldproject'
```

```
'oldproject' and everything in it will be deleted:

  oldproject                              512 files, 1.2 GB

It will be moved to the trash.
```

## Nothing here is skipped

Other commands walk past folders like `node_modules` and `.venv`. **`delete` does
not.** If it skipped them, you would be told the folder was emptied while
thousands of files quietly remained. So delete sees everything — and the preview
shows you the real size before you agree.

## When some files will not budge

`osql` cannot ask for admin rights. If a file needs them, it deletes what it can
and tells you the rest:

```
Moved 8 items to the trash.
Could not delete 2 items:
  Library/Caches/x    permission denied
Try running osql with sudo.
```

It keeps going rather than stopping halfway, so you always get a full report.

## Guards

**The root itself is protected.** `delete all from '/'` would empty your home
folder, so it is refused:

```bash
delete all from '/'
```

```
I won't empty '/' itself. Name a folder inside it, or add a where clause.
```

The same applies to your home folder. Naming a folder is fine
(`delete all from 'temp'`), and so is a filter
(`delete files from '/' where type = 'tmp'`).

**It also will not delete the folder you are standing in:**

```bash
delete folder '.'
```

```
'~/docs' is the folder you are in, so I won't delete it. Move somewhere else first with "cd ..".
```

Emptying its *contents* is allowed — `delete all from '.'` — with the usual
preview and confirmation.

<!-- nav -->

---

[← Creating files and folders](creating.md) · [All pages](README.md) · [Output →](output.md)
