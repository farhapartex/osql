<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Installation

Pick whichever line below suits you. All of them end with an `osql` you can run.

---

**On this page**

- [The quick way](#the-quick-way)
- [With Go](#with-go)
- [By hand](#by-hand)
- [From source](#from-source)
- [Check it worked](#check-it-worked)
- [What you need](#what-you-need)
- [First run](#first-run)
- [Updating](#updating)
- [Removing it](#removing-it)

## The quick way

```bash
curl -fsSL https://raw.githubusercontent.com/farhapartex/osql/main/install.sh | sh
```

This works out your operating system and processor, downloads the matching
release, **checks it against its published SHA256**, and puts `osql` in
`~/.local/bin`. It never asks for `sudo`.

If `~/.local/bin` is not on your `PATH`, the installer prints the exact line to
add.

### If you would rather read it first

Piping anything into `sh` means trusting it. If you would rather look before you
run, that is the sensible instinct:

```bash
curl -fsSL -o install.sh https://raw.githubusercontent.com/farhapartex/osql/main/install.sh
less install.sh
sh install.sh
```

### Installer options

```bash
sh install.sh --dir /usr/local/bin      # put it somewhere else
sh install.sh --version v0.1.0          # install a specific release
```

`OSQL_INSTALL_DIR` and `OSQL_VERSION` do the same thing if you prefer
environment variables.

## With Go

If you already have Go 1.26 or newer:

```bash
go install github.com/farhapartex/osql@latest
```

That puts `osql` in your `$GOBIN` (or `$GOPATH/bin`).

## By hand

Every release has a tarball for each platform on the
[releases page](https://github.com/farhapartex/osql/releases).

```bash
tar -xzf osql_v0.1.0_darwin_arm64.tar.gz
mv osql_v0.1.0_darwin_arm64/osql ~/.local/bin/
```

To check a download before you trust it, grab `checksums.txt` from the same
release:

```bash
sha256sum -c checksums.txt      # shasum -a 256 -c on macOS
```

Each tarball also carries the `LICENSE` and `README.md`.

## From source

Building from a clone is a contributor path rather than an install path — it
needs Go and gives you an unreleased binary. If that is what you want, see
[Development](development.md).

## Check it worked

```bash
osql --version
```

```
osql v0.1.0 (a1b2c3d)
```

If your shell says `command not found`, the folder you installed into is not on
your `PATH`:

```bash
export PATH="$PATH:$HOME/.local/bin"
```

Add that line to `~/.zshrc` or `~/.bashrc` to make it stick.

## What you need

- macOS or Linux, on Intel or ARM
- Go 1.26 or newer, **only** if you are building from source

## First run

The first time you start `osql` it makes a small folder at `~/.osql` for your
command history. Nothing leaves your machine — `osql` has no network code at
all. See [Your files](files.md) for what is in there.

## Updating

Run the installer again. It replaces the binary in place.

## Removing it

`osql` can remove itself:

```bash
osql uninstall
```

It works out where it was installed, shows you exactly what it is about to
delete, and waits for you to type `yes`:

```
This will remove:

  /Users/you/.local/bin/osql                    5.0 MB  the osql binary
  /Users/you/.osql                             12.0 KB  your history and system notes

Total: 5.0 MB

This cannot be undone. Type "yes" to go ahead, anything else to cancel.
uninstall>
```

Anything other than `yes` cancels and leaves everything alone.

### Options

```bash
osql uninstall --keep-data     # remove the program, keep ~/.osql
osql uninstall --yes           # do not ask, for scripts
```

`--keep-data` is the one to use if you plan to reinstall and want your command
history to survive.

### If it cannot remove itself

If you installed `osql` somewhere you do not own, such as `/usr/local/bin` on
many systems, it will not delete the file and it will never ask for your
password. It prints the line to run instead:

```
I don't have permission to remove '/usr/local/bin/osql', and I won't ask for it. Run this yourself:

  sudo rm '/usr/local/bin/osql'
```

If a package manager installed `osql`, use that package manager. `osql` will say
so and stop, because deleting the file behind its back leaves its records out of
step.

### By hand

If the binary is broken or missing, two lines do the same job:

```bash
rm ~/.local/bin/osql      # or wherever you put it
rm -rf ~/.osql            # your history and system notes
```

<!-- nav -->

---

[All pages](README.md) · [Queries →](queries.md)
