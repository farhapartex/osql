<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Error messages

Every message tries to say two things: what went wrong, and what to do about it.
This page lists them all with a fix.

## Paths

**`I couldn't find a folder at 'Documnets'. Check the path and try again.`**

The folder is not there. Check the spelling. Remember that paths start from your
home folder, so `/etc` means `~/etc`.

```bash
folders from '.'          # see what is actually there
```

---

**`'notes.txt' is a file, not a folder. Try: files from 'Documents'`**

You pointed at a file. Listing looks *inside* folders, so give it the folder that
holds the file. To print the file itself, use [`open`](opening.md).

---

**`'Documents' is a folder, not a file. Try: open 'Documents/notes.txt'`**

The opposite problem: `open` prints one file, so it cannot take a folder. To see
what is in it, use `files from 'Documents'`.

---

**`I couldn't find a file at 'notes.txt'. Check the path and try again.`**

Same as the missing-folder message, but from `open`. Check the spelling.

---

**`'photo.jpg' looks like a binary file, so I won't print it. open only shows text.`**

`open` refuses files that are not text, because printing them can break your
terminal. See [Opening files](opening.md#binary-files-are-refused).

---

**`I can only look inside '/Users/you'. '../etc' points outside it.`**

`..` cannot climb above where `osql` started. If you need somewhere else, start
it there:

```bash
osql --root /
```

---

**`I don't have permission to read 'Library'.`**

Your user cannot open that folder. Try `sudo`, or pick another folder.

## Getting the query shape right

**`Queries don't need "select" — start with what you want: files from 'Documents'`**

There is no `select` in `osql`. Drop it.

---

**`I need "files", "folders", or "all" to start — for example: files from 'Documents'`**

Your query has no subject. Say what you want first.

---

**`Use "files", not "file" — for example: files from 'Documents'`**

Always plural. `file` and `folder` are not accepted.

---

**`I can list "files", "folders", or "all" — not "filez". Did you mean "files"?`**

A typo. If the word is close to a real one, the message says which.

---

**`I need "from" before the folder — for example: files from 'Documents'`**

The word `from` is required between what you want and where to look.

---

**`I need a folder after "from" — for example: files from 'Documents'`**

The query stops at `from`. Add the folder.

---

**`I need a file after "open" — for example: open 'notes.txt'`**

You typed `open` with nothing after it.

---

**`I don't understand "junk" here. Try: files from 'Documents'`**

There is something extra on the end that does not belong.

---

**`The query ends after "where". I need more — for example: files from 'Documents' where name = 'notes.txt'`**

The query stops in the middle. Finish the condition.

## Filters

**`I don't know the field "extension". I understand: name, name_like, type, count(child)`**

That is not a field you can filter on. The message lists the ones you can. For
extensions the field is called `type`.

---

**`"name" only works with = and !=. For patterns use name_like: files from 'Documents' where name_like = '%report%'`**

You used `<`, `>`, or similar on a text field. Those only work on
`count(child)`. For loose name matching use `name_like`.

---

**`count(child) describes folders, not files. Try: folders from 'Documents' where count(child) > 10`**

Files have nothing inside them. Ask for `folders` instead.

---

**`count(child) needs a number — for example: count(child) > 10`**

You compared it to text. It counts things, so it needs a number.

## Typing mistakes

**`This quote is never closed: 'Documents — add a closing '`**

You opened a quote and never closed it. The message shows what came after it.
Note that `\'` is an apostrophe, not a closing quote — `'abc\'` is still open.

---

**`I don't know the escape "\q". I understand: \n, \t, \r, \\ and \'`**

You used a backslash followed by something that is not an escape. For a real
backslash, write `\\`.

---

**`count( needs a closing ) — for example: count(files) from 'Documents'`**

You opened `count(` and never closed it.

## Creating

**`'notes.txt' already exists. Nothing was changed.`**

`new` never overwrites. Pick a different name, or remove the old file yourself.

---

**`Use "file", not "files" — you make one thing at a time: new file 'notes.txt'`**

`new` takes the singular. Listing takes the plural.

---

**`I can make a "file" or a "folder" — not "thing".`**

`new` only knows those two words.

---

**`A folder can't hold data. Drop the data part, or use: new file 'notes.txt' data='hello'`**

`data=` only works with `new file`.

---

**`data needs a value in quotes — for example: data='hello there'`**

You wrote `data=` with nothing after it.

---

**`I couldn't create 'locked/notes.txt': permission denied`**

The folder will not let you write to it.

## Summary

**`"with" needs "skipped" — for example: summary from 'Documents' recursive with skipped`**

You wrote `with` on its own. The only thing that follows it is `skipped`.

---

**`I only know "with skipped", not "with everything".`**

Same idea — `with skipped` is the one option.

## Deleting

**`I won't empty '/Users/you' itself. Name a folder inside it, or add a where clause.`**

Deleting everything in your home folder is refused. Name a folder inside it, or
narrow it with `where`.

---

**`'reports' is a folder, not a file. Try: delete folder 'reports'`**

You used the wrong one of the pair. The message gives you the right command.

---

**`I can't delete "thing". Try "delete file", "delete folder", or "delete files from".`**

`delete` takes `file`, `folder`, `files`, `folders` or `all`.

---

**`'x' is on another disk, so it can't go to the trash. Add "permanently" to delete it for good.`**

The trash lives in your home folder, and files cannot be moved across disks. Use
`permanently` if you are sure.

---

**`I couldn't delete 'x': permission denied`**

The file needs admin rights. osql cannot ask for them — try running it with
`sudo`.

## Outcomes, not errors

These two are normal answers, not problems:

**`No files matched.`** — the query worked; nothing fit the filter.

**`'Documents' is empty.`** — the folder is real and has nothing in it.

## One rough edge

A typo in a *built-in command* gets the query message rather than a suggestion:

```bash
helpp
```

```
I can list "files", "folders", or "all" — not "helpp".
```

It should suggest `help`. Type `help` to see the list of commands.
