<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Installed apps

To see what is installed on your machine, ask for `apps`.

```bash
apps
```

```
NAME           VERSION         SOURCE    MODIFIED
Calculator     10.16           macos     2023-07-11 14:56
Claude         1.30096.5       system    2026-08-14 22:05
Google Chrome  151.0.7922.174  system    2024-05-10 22:45
OrbStack       1.6.1_17010     homebrew  2024-06-25 17:20
Safari         26.0            macos     2026-06-13 10:22

81 apps
```

## There is no path

Every other query takes a folder. `apps` does not, because apps are not in one
folder you pick — osql already knows the places your system puts them.

```bash
apps
```

If you add a path, osql tells you it is not needed.

```
"apps" already looks everywhere your system installs apps, so it needs no path. Try: apps
```

`apps` is also never recursive. Looking inside an app would list the small
helper programs it ships with as if they were apps of their own.

## What SOURCE means

The source tells you where the app came from, which is usually what you want to
know before you touch it.

| Source | Meaning |
| --- | --- |
| `macos` | Came with macOS |
| `system` | Installed for everyone on the machine |
| `user` | Installed for you only |
| `homebrew` | Installed by Homebrew as a cask |
| `homebrew-cli` | A Homebrew command-line tool |
| `apt` | Installed by your package manager (Linux) |
| `flatpak` | Installed as a Flatpak (Linux) |
| `snap` | Installed as a Snap (Linux) |

## Command-line tools are hidden by default

Homebrew installs a lot of small command-line tools. On a normal machine they
outnumber real apps, so `apps` leaves them out and says so:

```
81 apps
Not showing 129 command-line tools. Add "where source = 'homebrew-cli'" to see them.
```

Ask for them when you want them:

```bash
apps where source = 'homebrew-cli'
```

## How much disk each app uses

Add `with size`.

```bash
apps with size
```

```
NAME           VERSION         SOURCE    SIZE      MODIFIED
Among Us       —               system    958.8 MB  2024-12-23 21:54
App Store      3.0             macos     15.5 MB   2023-07-11 14:56
Google Chrome  151.0.7922.174  system    2.0 GB    2024-05-10 22:45

81 apps, 29.2 GB on disk
```

It is a separate word because it is the slow part. An app is a folder with
thousands of files inside, so measuring them all means reading every one —
on a normal machine that is a second or two, against a few hundredths for a
plain `apps`. You only pay it when you ask.

**Filter first and it stays quick.** osql measures what is left after the
`where` clause, not everything:

```bash
apps with size where name_like = '%Chrome%'
```

That measures one app, so it returns straight away.

`with size` goes before `where`:

```bash
apps with size where source = 'homebrew'
```

A size of `—` means osql could not read inside that app. It is left out of the
total rather than counted as zero.

Sizes add up the real files inside the app. Shortcuts pointing outside the app
are not followed, so nothing is counted twice.

## Filtering

`apps` takes the same `where` clause as everything else.

```bash
apps where name_like = '%Chrome%'
apps where source = 'macos'
apps where version_like = '1.%'
apps where id = 'com.apple.Safari'
apps where name_like = '%Chrome%' and source = 'system'
```

The fields you can filter on:

| Field | Example |
| --- | --- |
| `name` | `apps where name = 'Safari'` |
| `name_like` | `apps where name_like = '%Chrome%'` |
| `version` | `apps where version = '18.2'` |
| `version_like` | `apps where version_like = '1.%'` |
| `source` | `apps where source = 'homebrew'` |
| `id` | `apps where id = 'com.apple.Safari'` |
| `id_like` | `apps where id_like = 'com.apple.%'` |

Values are case-sensitive, so `'%chrome%'` does not match `Google Chrome`. Match
the capital letters you see in the NAME column.

`type` and `count(child)` are about files and folders, so they do not work here —
and `version`, `source`, and `id` do not work on files.

## Counting

```bash
count(apps)
```

```
WHAT  COUNT
apps  81
```

A count has no size column, so `count(apps) with size` is refused rather than
spending a second measuring apps for a number you did not ask for.

## Why a version is sometimes missing

A missing version shows as `—`. On macOS a handful of Apple's own apps store
their details in a compact format osql does not read yet, so their name and date
are right but the version is blank. Everything else reports its real version.

## osql will not uninstall

There is no way to delete an app with osql, on purpose.

```
I won't uninstall apps — removing one properly also means its settings and
background helpers, which I can't do safely. Use your system's own uninstaller.
```

Removing the app folder alone leaves settings and background helpers behind, so
osql does not pretend it can do the job.
