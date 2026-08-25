# Your files

`osql` keeps two small text files in one folder:

```bash
~/.osql/system.txt     # details about your machine
~/.osql/history.txt    # the commands you have typed
```

The folder is created the first time you start `osql`. To remove every trace:

```bash
rm -rf ~/.osql
```

## Nothing is sent anywhere

`osql` has no network code at all. It cannot phone home, because there is nothing
in it that can make a network request. Both files stay on your machine.

## system.txt

Written once, when the folder is created. Plain `key: value` lines:

```bash
cat ~/.osql/system.txt
```

```
version: v0.1.0
commit: a1b2c3d
created_at: 2026-08-25T12:00:00Z
os: darwin
arch: arm64
kernel: 22.6.0
cpus: 8
go: go1.26.7
hostname: my-laptop
user: you
uid: 501
home: /Users/you
shell: /bin/zsh
```

This is for you, when something behaves oddly and you want to say what you are
running on. It does include your username and computer name, so glance at it
before pasting it into a public bug report.

To refresh it:

```bash
osql init --reinit
```

## history.txt

One line per command, in the order you typed them. Invalid queries are saved
too — a typo is usually the line you want to fix and try again.

```bash
cat ~/.osql/history.txt
```

The file keeps the most recent **10,000** lines. Older ones are dropped when
`osql` starts.

To see it from inside the shell:

```bash
history
```

To empty it:

```bash
history clear
```

To skip saving for one session:

```bash
osql --no-history
```

With `--no-history`, the file is not even created.

## Permissions

| Path | Mode | Why |
|---|---|---|
| `~/.osql/` | `0700` | only you can open the folder |
| `~/.osql/system.txt` | `0644` | nothing private in it |
| `~/.osql/history.txt` | `0600` | it lists folders you have looked at |

`history.txt` is stricter because your queries mention paths, and those can be
telling on a shared machine.
