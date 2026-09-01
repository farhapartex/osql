<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Filtering

Add `where` to narrow down what you get back.

```bash
files from 'Documents' where type = 'txt'
```

---

**On this page**

- [What you can filter on](#what-you-can-filter-on)
- [By name](#by-name)
- [By pattern](#by-pattern)
- [By type](#by-type)
- [By size](#by-size)
- [By how full a folder is](#by-how-full-a-folder-is)
- [Combining filters](#combining-filters)
- [Filtering while going deep](#filtering-while-going-deep)
- [Spacing](#spacing)

## What you can filter on

| Field | Looks at | Works with |
|---|---|---|
| `name` | the whole file name | `=` `!=` |
| `name_like` | the name, with wildcards | `=` `!=` |
| `type` | the file extension | `=` `!=` |
| `size` | how big a file is | `=` `!=` `<` `>` `<=` `>=` |
| `count(child)` | how many items are inside a folder | `=` `!=` `<` `>` `<=` `>=` |

Asking for [apps](apps.md) instead of files swaps the last three for `version`,
`version_like`, `source`, `id`, and `id_like`. osql tells you when a field does
not fit what you asked for, and lists the ones that do.

## By name

```bash
files from 'Documents' where name = 'notes.txt'
files from 'Documents' where name != 'secret.txt'
```

`name` is an exact match. For anything looser, use `name_like`.

## By pattern

Use `%` where you do not care what comes next.

```bash
files from 'Documents' where name_like = '%report%'    # contains "report"
files from 'Documents' where name_like = 'report%'     # starts with "report"
files from 'Documents' where name_like = '%report'     # ends with "report"
files from 'Documents' where name_like = '%.log'       # any .log file
```

`*` works the same way if that is what your fingers type:

```bash
files from 'Documents' where name_like = '*report*'
```

## By type

`type` is the bit after the last dot. The dot itself is optional:

```bash
files from 'Documents' where type = 'txt'
files from 'Documents' where type = '.txt'    # same thing
```

Files with no extension have an empty type, and folders have the type `folder`:

```bash
all from 'Documents' where type = 'folder'
```

Whatever you see in the TYPE column, you can paste back into a query.

## By size

`size` is how big a file is, in bytes unless you add a unit:

```bash
files from 'Downloads' where size > 100mb
files from 'Documents' where size < 1kb
files from '~' recursive where size >= 1gb
```

The units are `b`, `kb`, `mb`, `gb` and `tb`, and they are not case
sensitive, so `10MB` and `10mb` are the same. Each step is 1024, matching the
sizes in the SIZE column, so a file the table shows as `1.2 MB` does match
`size > 1mb`. Decimals work too: `size > 1.5gb`.

Attach the unit to the number. If you would rather leave a space, put the whole
value in quotes:

```bash
files from 'Downloads' where size > 100mb        # fine
files from 'Downloads' where size > '100 mb'     # also fine
```

`size` is about files. Folders show `—` in the SIZE column, because working out
how big a folder is means adding up everything inside it, so asking
`folders … where size > …` tells you the field does not fit.

## By how full a folder is

`count(child)` counts the items directly inside a folder.

```bash
folders from 'src' where count(child) > 10     # crowded folders
folders from 'src' where count(child) = 0      # empty folders
folders from '~' where count(child) <= 2       # nearly empty
```

This one only makes sense for folders. Asking for it with `files` gives you a
message saying so.

## Combining filters

Join conditions with `and`. Every condition must be true.

```bash
files from 'src' where name_like = 'test_%' and type = 'go'
files from 'Documents' where type = 'pdf' and name_like = '%2026%'
```

`or` is not supported yet.

## Filtering while going deep

`where` and `recursive` work together:

```bash
files from '~' recursive where name = 'notes.txt'
files from 'src' recursive where type = 'go' and name_like = '%_test%'
```

## Spacing

Spaces around `=` are optional:

```bash
files from 'Documents' where type='txt'
```

<!-- nav -->

---

[← Queries](queries.md) · [All pages](README.md) · [Counting →](counting.md)
