# Counting

Sometimes you only want the number. Wrap what you are asking for in `count(...)`.

```bash
count(files) from 'Documents'
```

```
WHAT   COUNT
files  52
```

## The three forms

```bash
count(files) from 'Documents'
count(folders) from 'Documents'
count(all) from 'Documents'
```

`count(all)` gives you **two rows**, because files and folders are different
things and one combined number hides which is which:

```
WHAT     COUNT
files    52
folders  29
```

## Same filters as a listing

Counting takes `recursive` and `where` exactly like a normal query:

```bash
count(files) from 'src' recursive where type = 'go'
count(folders) from 'Documents' where count(child) = 0
count(all) from '~' recursive
```

The number is always exactly how many rows the same query without `count(...)`
would have printed. These two always agree:

```bash
files from 'Documents' where type = 'txt'
count(files) from 'Documents' where type = 'txt'
```

## Why it is fast

Counting skips reading each file's size and date, because a total does not need
them. On a folder with 100,000 files that saves 100,000 system calls. Counting a
big tree is noticeably quicker than listing it.

## Zero is a real answer

If nothing matches you get `0`, not an error:

```bash
count(files) from 'Documents' where type = 'xyz'
```

```
WHAT   COUNT
files  0
```

## Next

- [Filtering](filtering.md) — the full list of things you can filter on
- [Output](output.md) — reading the normal table
