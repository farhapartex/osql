<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Shell and flags

The shell is where you type queries. This page covers everything around
them: the commands that are not queries, the keys that edit your line, and the
flags you can start with.

---

**On this page**

- [Starting and leaving](#starting-and-leaving)
- [Built-in commands](#built-in-commands)
- [Command-line flags](#command-line-flags)
- [Setting up ahead of time](#setting-up-ahead-of-time)
- [Editing the line you are typing](#editing-the-line-you-are-typing)
- [Known limits](#known-limits)

## Starting and leaving

```bash
osql
```

To leave, type `exit` or `quit`, or press `Ctrl+D`.

## Built-in commands

These are not queries — they are things the shell itself does.

| Command | What it does |
|---|---|
| `cd <path>` | move to another folder; `cd` alone goes home, `cd -` goes back |
| `pwd` | show the folder you are in |
| `help` | list everything you can type |
| `history` | show your recent commands, numbered |
| `history clear` | empty the history file |
| `clear` | clear the screen |
| `exit` / `quit` | leave the shell |

```bash
cd Documents
pwd
help
history
history clear
```

## Command-line flags

| Flag | What it does |
|---|---|
| `--dir <path>` | start in `<path>` instead of the folder you ran osql from |
| `--no-history` | do not save this session's commands |
| `--version` | print the version and exit |
| `--help` | print a short usage message |

```bash
osql --dir /var/log
osql --no-history
osql --version
```

## Setting up ahead of time

```bash
osql init
```

This creates `~/.osql` and writes down some details about your machine. You do
not need to run it — `osql` does the same thing on its first start. It is there
for install scripts.

To rewrite the machine details after a version change:

```bash
osql init --reinit
```

## Editing the line you are typing

The prompt is a real line editor. Arrow keys move the cursor, and up and down
walk through your history.

| Key | Does |
|---|---|
| ← → | move the cursor one character |
| ↑ ↓ | previous / next command from your history |
| Home / End | jump to the start or end of the line |
| Backspace | delete the character before the cursor |
| Delete | delete the character under the cursor |
| Ctrl+A / Ctrl+E | start / end of line |
| Ctrl+B / Ctrl+F | back / forward one character |
| Alt+← / Alt+→ | move a whole word |
| Ctrl+W | delete the word before the cursor |
| Ctrl+K | delete from the cursor to the end |
| Ctrl+U | delete from the start to the cursor |
| Ctrl+L | clear the screen |
| Ctrl+C | throw away the line you are typing, or stop a query that is running |
| Ctrl+D | leave the shell (or delete forward, if the line is not empty) |

Pressing ↑ then editing the recalled line works as you would expect. Pressing ↓
back past the newest entry brings back whatever you had half-typed.

### Stopping a query that is taking too long

A recursive search over a large folder can take a while. Press **Ctrl+C** to
stop it. `osql` tells you it stopped, says how much it had found, and gives you
the prompt back:

```
osql > files from '~' recursive
^C
Stopped after 4193 matches. Narrow the folder or add a where clause to make it quicker.
osql >
```

The shell itself keeps running, so nothing is lost. A `delete` is the one thing
Ctrl+C will not cut in half: once you have typed `yes` and the deleting has
started, it finishes. Press Ctrl+C before that and nothing is deleted.

### When it is not available

If `osql` is not attached to a terminal — piped input, a script, a CI job — it
falls back to reading plain lines. Everything still works, there is just no
cursor to move:

```bash
printf "files from 'Documents'
exit
" | osql
```

## Known limits

**No tab completion.** Tab does nothing yet.

**No paging.** `open` on a huge file prints all of it.

**Nothing is overwritten.** `new` refuses rather than replace an existing file,
and `delete` always asks before removing anything. See [deleting](deleting.md).

**osql cannot ask for admin rights.** If a file needs `sudo`, osql reports that
it could not delete it rather than prompting for a password.

<!-- nav -->

---

[← Output](output.md) · [All pages](README.md) · [Error messages →](errors.md)
