<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Shell and flags

## Starting and leaving

```bash
osql
```

To leave, type `exit` or `quit`, or press `Ctrl+D`.

## Built-in commands

These are not queries — they are things the shell itself does.

| Command | What it does |
|---|---|
| `help` | list everything you can type |
| `history` | show your recent commands, numbered |
| `history clear` | empty the history file |
| `clear` | clear the screen |
| `exit` / `quit` | leave the shell |

```bash
help
history
history clear
```

## Command-line flags

| Flag | What it does |
|---|---|
| `--root <path>` | look inside `<path>` instead of your home folder |
| `--no-history` | do not save this session's commands |
| `--version` | print the version and exit |
| `--help` | print a short usage message |

```bash
osql --root /
osql --root /var/log
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

## Known limits

**Arrow keys do not recall commands yet.** Pressing the up arrow prints `^[[A`
instead of your last query. Your commands *are* saved, so `history` shows them —
you just cannot scroll back through them.

The reason is that proper arrow-key editing needs raw terminal control, which is
normally handled by an outside library, and `osql` deliberately has none. It is
on the list.

**Ctrl+C leaves the shell** instead of cancelling the line you are typing. Same
reason.

**Nothing is overwritten.** `new` refuses rather than replace an existing file,
and `delete` always asks before removing anything. See [deleting](deleting.md).

**osql cannot ask for admin rights.** If a file needs `sudo`, osql reports that
it could not delete it rather than prompting for a password.

**No paging.** `open` on a huge file prints all of it. There is no `less`-style
pager yet.

## Next

- [Your files](files.md) — what gets saved in `~/.osql`
- [Error messages](errors.md) — what a message means
