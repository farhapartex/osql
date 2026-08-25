# Opening files

`open` prints what is inside a text file, the way `cat` does.

```bash
open 'notes.txt'
```

```
first line
second line
```

## Paths work the same way

Same rules as every other query — everything is inside your home folder, and
these all mean one file:

```bash
open 'Documents/notes.txt'
open '/Documents/notes.txt'
open '~/Documents/notes.txt'
```

Quotes are optional when there are no spaces:

```bash
open Documents/notes.txt
```

## It only takes a path

No `where`, no `recursive`. `open` prints one file and stops.

```bash
open 'notes.txt' where name = 'x'
```

```
I don't understand "where" here. Try: files from 'Documents'
```

## Folders are refused

`open` needs a file. Point it at a folder and it says so:

```bash
open 'Documents'
```

```
'Documents' is a folder, not a file. Try: open 'Documents/notes.txt'
```

To see what is in a folder, list it instead:

```bash
files from 'Documents'
```

## Binary files are refused

Printing a program or an image into your terminal fills it with junk and can
leave it in a broken state. So `osql` checks first:

```bash
open 'bin/osql'
```

```
'bin/osql' looks like a binary file, so I won't print it. open only shows text.
```

Nothing is printed before the message. If a file turns out to be binary partway
through, `osql` stops there — so you may see the text part before the message.

This is the one place `open` deliberately does *less* than `cat`.

## Other messages

```bash
open 'nope.txt'
```

```
I couldn't find a file at 'nope.txt'. Check the path and try again.
```

```bash
open
```

```
I need a file after "open" — for example: open 'notes.txt'
```

If you cannot read the file:

```
I don't have permission to read 'locked.txt'.
```

## Small details

- The file is printed exactly as stored. Windows line endings and unicode come
  through unchanged.
- If the file has no newline at the end, `osql` adds one so the next prompt
  starts on a fresh line.
- An empty file prints nothing at all.
- Big files are streamed, not loaded into memory, so opening a large log will not
  slow your machine down. It will still fill your screen — `osql` has no paging
  yet.

## Next

- [Queries](queries.md) — listing folders
- [Error messages](errors.md) — the full list of messages
