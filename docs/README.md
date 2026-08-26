<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Documentation

Everything there is to read about `osql`, in the order it makes sense to read it.

New here? Start with [Installation](installation.md), then [Queries](queries.md).
Looking something up? Jump straight to the page you need.

---

## Getting started

| Page | What it covers |
| --- | --- |
| [Installation](installation.md) | What you need, how to build it, and how to put it on your PATH |
| [Queries](queries.md) | Your first query, the four things you can ask for, and how paths work |
| [Shell and flags](shell.md) | Built-in commands, editing keys, and command-line options |

## Asking questions

| Page | What it covers |
| --- | --- |
| [Filtering](filtering.md) | The `where` clause — match by name, type, wildcards, and size |
| [Counting](counting.md) | Getting a number back instead of a list |
| [Summary](summary.md) | A folder at a glance: totals, types, and the biggest files |
| [Installed apps](apps.md) | Listing the apps on your machine, with versions and disk usage |
| [Opening files](opening.md) | Printing what is inside a text file |

## Changing things

| Page | What it covers |
| --- | --- |
| [Creating](creating.md) | Making new files and folders, with content |
| [Deleting](deleting.md) | Removing files and folders, and the safety checks around it |

## Reference

| Page | What it covers |
| --- | --- |
| [Output](output.md) | Reading the table, the size column, and the dash |
| [Error messages](errors.md) | Every message osql can show, what it means, and how to fix it |
| [Your files](files.md) | What osql keeps on your machine, and what it never touches |
| [Development](development.md) | Building, testing, and how the project is laid out |

---

## Quick reference

Every kind of query, on one screen:

```bash
files from 'Documents'                        # list files
folders from 'Documents'                      # list folders
all from 'Documents'                          # list both
files from 'Documents' recursive              # include subfolders
files from 'Documents' where type = 'pdf'     # filter

count(files) from 'Documents'                 # how many
summary from 'Documents' recursive            # at a glance

apps                                          # installed apps
apps with size                                # with disk usage
summary apps                                  # apps at a glance

open 'Documents/notes.txt'                    # print a text file
new file 'notes.txt' data='hello'             # create
new folder 'reports'                          # create
delete file 'notes.txt'                       # move to trash
```

Built-in commands: `help`, `history`, `clear`, `exit`.

---

**Something missing or unclear?** The [error messages](errors.md) page lists every
message osql can print, so if the shell said something you did not expect, that is
the fastest place to look.
