<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Creating files and folders

`new` makes a file or a folder. Say which one, then where.

```bash
new file 'notes.txt'
new folder 'reports'
```

```
Created 'notes.txt'
```

Note the **singular** word here. You list many things (`files from …`) but you
make one thing at a time (`new file …`).

---

**On this page**

- [Putting text in the file](#putting-text-in-the-file)
- [Missing folders are created for you](#missing-folders-are-created-for-you)
- [Nothing is ever overwritten](#nothing-is-ever-overwritten)
- [Paths work the same as everywhere else](#paths-work-the-same-as-everywhere-else)
- [Other messages](#other-messages)
- [What gets set](#what-gets-set)

## Putting text in the file

Add `data=` to create the file with something already in it:

```bash
new file 'greeting.txt' data='hello hello line testing'
```

Read it back:

```bash
open 'greeting.txt'
```

```
hello hello line testing
```

Without `data=` you get an empty file, the way `touch` does.

### Multiple lines and quotes

Use `\n` for a line break and `\'` for an apostrophe:

```bash
new file 'notes.txt' data='line one\nline two\nline three'
new file 'diary.txt' data='it\'s working'
```

```bash
open 'notes.txt'
```

```
line one
line two
line three
```

The full list is `\n` (new line), `\t` (tab), `\r`, `\\` (one backslash) and
`\'` (apostrophe). Anything else after a backslash is refused, so a typo does not
end up inside your file:

```bash
new file 'x.txt' data='a\qb'
```

```
I don't know the escape "\q". I understand: \n, \t, \r, \\ and \'
```

The text is otherwise written exactly as you typed it — no newline is added at
the end. For anything long, make the file and edit it in your editor.

Folders hold files, not text, so `data` on a folder is refused:

```bash
new folder 'reports' data='hello'
```

```
A folder can't hold data. Drop the data part, or use: new file 'notes.txt' data='hello'
```

## Missing folders are created for you

If the path needs folders that do not exist yet, `osql` makes them:

```bash
new file 'reports/2026/q4/summary.txt'
```

```
Created 'reports/2026/q4/summary.txt'
  also created: reports, reports/2026, reports/2026/q4
```

The second line lists every folder it had to make. Read it — that is how you
catch a typo. If you meant `reports` and typed `reprots`, you will see a new
`reprots` folder appear in that list.

Folders that were already there are not listed.

This works the same for `new folder`:

```bash
new folder 'a/b/c'
```

## Nothing is ever overwritten

If something is already at that path, `osql` stops and changes nothing:

```bash
new file 'notes.txt'
```

```
'notes.txt' already exists. Nothing was changed.
```

This holds even with `data=`. Your file is safe:

```bash
new file 'important.txt' data='oops'
```

```
'important.txt' already exists. Nothing was changed.
```

There is no `delete` yet and no undo, so `new` refuses rather than risk your
content.

## Paths work the same as everywhere else

Everything is created inside your home folder, and these all mean one path:

```bash
new file 'Documents/notes.txt'
new file '/tmp/notes.txt'
new file '~/Documents/notes.txt'
new file '../sibling/notes.txt'
```

Anywhere you can write, osql can create. If you cannot write there, it says so:

```bash
new file '/notes.txt'
```

```
I couldn't create '/notes.txt': permission denied
```

## Other messages

```bash
new files 'a.txt'
```

```
Use "file", not "files" — you make one thing at a time: new file 'notes.txt'
```

```bash
new thing 'a'
```

```
I can make a "file" or a "folder" — not "thing".
```

```bash
new file
```

```
I need a path after "new file" — for example: new file 'notes.txt'
```

## What gets set

New files are readable and writable by you, readable by others (`0644`). New
folders are `0755`. These are the same defaults your shell uses.

<!-- nav -->

---

[← Opening files](opening.md) · [All pages](README.md) · [Deleting →](deleting.md)
