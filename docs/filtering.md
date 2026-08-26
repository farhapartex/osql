<img src="../osql_chevron.png" alt="osql" width="140" align="right">

# Filtering

Add `where` to narrow down what you get back.

```bash
files from 'Documents' where type = 'txt'
```

## The four things you can filter on

| Field | Looks at | Works with |
|---|---|---|
| `name` | the whole file name | `=` `!=` |
| `name_like` | the name, with wildcards | `=` `!=` |
| `type` | the file extension | `=` `!=` |
| `count(child)` | how many items are inside a folder | `=` `!=` `<` `>` `<=` `>=` |

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

## Next

- [Counting](counting.md) — the same filters, but you get a number
- [Error messages](errors.md) — if a filter is rejected
