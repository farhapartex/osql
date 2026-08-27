#!/usr/bin/env bash
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$ROOT/bin/osql"

PASS=0
FAIL=0
FAILURES=()

if [ -t 1 ]; then
  GREEN=$'\033[32m'; RED=$'\033[31m'; DIM=$'\033[2m'; BOLD=$'\033[1m'; OFF=$'\033[0m'
else
  GREEN=""; RED=""; DIM=""; BOLD=""; OFF=""
fi

section() {
  printf '\n%s%s%s\n' "$BOLD" "$1" "$OFF"
}

pass() {
  PASS=$((PASS + 1))
  printf '  %s✔%s %s\n' "$GREEN" "$OFF" "$1"
}

fail() {
  FAIL=$((FAIL + 1))
  FAILURES+=("$1")
  printf '  %s✘%s %s\n' "$RED" "$OFF" "$1"
  printf '    %s%s%s\n' "$DIM" "$2" "$OFF"
  if [ -n "${3:-}" ]; then
    printf '%s\n' "$3" | sed "s/^/      ${DIM}|${OFF} /"
  fi
}

setup() {
  WORK="$(mktemp -d)"
  export HOME="$WORK/home"
  FIXTURE="$HOME"

  mkdir -p "$FIXTURE"/{docs,src/big,src/one,empty_ish,nested/deep}

  printf 'notes' > "$FIXTURE/docs/notes.txt"
  printf 'pdf' > "$FIXTURE/docs/report.pdf"
  printf 'q4' > "$FIXTURE/docs/q4-report.txt"
  printf 'secret' > "$FIXTURE/docs/secret.txt"
  printf 'make' > "$FIXTURE/docs/Makefile"

  printf 'lex' > "$FIXTURE/src/test_lexer.go"
  printf 'parse' > "$FIXTURE/src/test_parser.go"
  printf 'help' > "$FIXTURE/src/helper.go"
  printf 'only' > "$FIXTURE/src/one/only.txt"
  for i in $(seq 1 11); do printf 'x' > "$FIXTURE/src/big/f$i.txt"; done

  mkdir -p "$FIXTURE/truly_empty"
  mkdir -p "$FIXTURE/proj/node_modules/pkg" "$FIXTURE/proj/.venv/lib" "$FIXTURE/onlydirs/a" "$FIXTURE/onlydirs/b"
  printf 'code' > "$FIXTURE/proj/main.go"
  mkdir -p "$FIXTURE/wide"
  printf 'x' > "$FIXTURE/wide/a-very-long-filename-that-should-be-truncated-in-the-summary-table.txt"
  printf 'yy' > "$FIXTURE/wide/short.txt"
  printf 'zzz' > "$FIXTURE/wide/日本語のファイル名前です.md"
  printf 'readme' > "$FIXTURE/proj/README.md"
  head -c 4096 /dev/zero | tr '\0' 'n' > "$FIXTURE/proj/node_modules/pkg/big.js"
  head -c 2048 /dev/zero | tr '\0' 'v' > "$FIXTURE/proj/.venv/lib/mod.py"
  printf 'keep' > "$FIXTURE/empty_ish/.keep"
  printf 'far' > "$FIXTURE/nested/deep/far.txt"
  printf 'app' > "$FIXTURE/app.log"
  mkdir -p "$FIXTURE/text"
  printf 'line one\nline two\n' > "$FIXTURE/text/readme.txt"
  printf 'no trailing newline' > "$FIXTURE/text/bare.txt"
  : > "$FIXTURE/text/empty.txt"
  printf '\x7fELF\x00\x01\x02binary' > "$FIXTURE/text/prog.bin"
  printf '\xe6\x97\xa5\xe6\x9c\xac\xe8\xaa\x9e\n' > "$FIXTURE/text/unicode.txt"
  printf 'err' > "$FIXTURE/error.log"

  head -c 1023 /dev/zero | tr '\0' 'a' > "$FIXTURE/size_1023.bin"
  head -c 1024 /dev/zero | tr '\0' 'a' > "$FIXTURE/size_1024.bin"
  head -c 1048575 /dev/zero | tr '\0' 'a' > "$FIXTURE/size_1048575.bin"

  mkdir -p "$HOME/Documents"
  printf 'home' > "$HOME/Documents/home_notes.txt"
}

teardown() {
  chmod -R u+rwx "$WORK" 2>/dev/null
  rm -rf "$WORK"
}

osql_script() {
  printf '%s' "$1" | (cd "$FIXTURE" && "$BIN") 2>&1 | tail -n +2 | sed 's/osql > //g; s/delete> //g'
}

expect_script_contains() {
  local name="$1" script="$2"; shift 2
  local out; out="$(osql_script "$script")"
  local missing=()
  for want in "$@"; do
    printf '%s' "$out" | grep -qF -- "$want" || missing+=("$want")
  done
  if [ ${#missing[@]} -eq 0 ]; then
    pass "$name"
  else
    fail "$name" "missing: ${missing[*]}" "$out"
  fi
}

file_gone() {
  local name="$1" path="$2"
  if [ -e "$FIXTURE/$path" ]; then
    fail "$name" "$path is still there"
  else
    pass "$name"
  fi
}

file_kept() {
  local name="$1" path="$2"
  if [ -e "$FIXTURE/$path" ]; then
    pass "$name"
  else
    fail "$name" "$path was deleted"
  fi
}

file_mode() {
  mode="$(stat -c '%a' "$1" 2>/dev/null || true)"
  case "$mode" in
    [0-7][0-7][0-7]|[0-7][0-7][0-7][0-7]) printf '%s' "$mode"; return 0 ;;
  esac
  mode="$(stat -f '%Lp' "$1" 2>/dev/null || true)"
  case "$mode" in
    [0-7][0-7][0-7]|[0-7][0-7][0-7][0-7]) printf '%s' "$mode"; return 0 ;;
  esac
  printf 'unknown'
}

expect_apps_query() {
  local name="$1" q="$2"
  local out; out="$(osql "$q")"
  if printf '%s' "$out" | grep -qE '^NAME +VERSION'; then
    pass "$name"
  elif printf '%s' "$out" | grep -qF "No apps matched."; then
    pass "$name (no matching apps on this host)"
  elif printf '%s' "$out" | grep -qF "I didn't find any installed apps."; then
    pass "$name (no apps on this host)"
  else
    fail "$name" "query: $q -- neither a table nor a no-match message" "$out"
  fi
}

osql_raw() {
  printf '%s\nexit\n' "$1" | (cd "$FIXTURE" && "$BIN") 2>&1
}

osql() {
  osql_raw "$1" | tail -n +2 | sed 's/osql[^>]*> //g'
}

expect_contains() {
  local name="$1" q="$2"; shift 2
  local out; out="$(osql "$q")"
  local missing=()
  for want in "$@"; do
    printf '%s' "$out" | grep -qF -- "$want" || missing+=("$want")
  done
  if [ ${#missing[@]} -eq 0 ]; then
    pass "$name"
  else
    fail "$name" "query: $q -- missing: ${missing[*]}" "$out"
  fi
}

expect_absent() {
  local name="$1" q="$2"; shift 2
  local out; out="$(osql "$q")"
  local present=()
  for bad in "$@"; do
    printf '%s' "$out" | grep -qF -- "$bad" && present+=("$bad")
  done
  if [ ${#present[@]} -eq 0 ]; then
    pass "$name"
  else
    fail "$name" "query: $q -- should not contain: ${present[*]}" "$out"
  fi
}

expect_line() {
  local name="$1" q="$2" want="$3"
  local out; out="$(osql "$q")"
  if printf '%s' "$out" | grep -qxF -- "$want"; then
    pass "$name"
  else
    fail "$name" "query: $q -- no line equal to: $want" "$out"
  fi
}

expect_names() {
  local name="$1" q="$2" want="$3"
  local out got
  out="$(osql "$q")"
  got="$(printf '%s' "$out" | awk '/^NAME /{h=1;next} /files$/{h=0} h && NF {print $1}' | sort | tr '\n' ' ' | sed 's/ $//')"
  if [ "$got" = "$want" ]; then
    pass "$name"
  else
    fail "$name" "query: $q" "expected: $want
     actual: $got"
  fi
}

expect_names_from_count() {
  local name="$1" q="$2" want="$3"
  local out got
  out="$(osql "$q")"
  got="$(printf '%s' "$out" | awk '/^WHAT/{h=1;next} h && NF {printf "%s %s|", $1, $2}' | sed 's/|$//')"
  if [ "$got" = "$want" ]; then
    pass "$name"
  else
    fail "$name" "query: $q" "expected: $want
     actual: $got"
  fi
}

expect_shell_contains() {
  local name="$1" script="$2"; shift 2
  local out; out="$(printf '%s' "$script" | (cd "$FIXTURE" && "$BIN") 2>&1)"
  local missing=()
  for want in "$@"; do
    printf '%s' "$out" | grep -qF -- "$want" || missing+=("$want")
  done
  if [ ${#missing[@]} -eq 0 ]; then
    pass "$name"
  else
    fail "$name" "missing: ${missing[*]}" "$out"
  fi
}

expect_cmd() {
  local name="$1" want_code="$2"; shift 2
  local out code
  out="$(cd "$FIXTURE" && "$@" 2>&1)"; code=$?
  if [ "$code" -eq "$want_code" ]; then
    pass "$name"
  else
    fail "$name" "exit code $code, want $want_code" "$out"
  fi
}

expect_cmd_contains() {
  local name="$1" want="$2"; shift 2
  local out; out="$(cd "$FIXTURE" && "$@" 2>&1)"
  if printf '%s' "$out" | grep -qF -- "$want"; then
    pass "$name"
  else
    fail "$name" "missing: $want" "$out"
  fi
}

main() {
  printf '%sosql end-to-end%s  %s(real binary, real filesystem)%s\n' "$BOLD" "$OFF" "$DIM" "$OFF"

  if [ ! -x "$BIN" ]; then
    printf '\n%sbuilding%s\n' "$DIM" "$OFF"
    (cd "$ROOT" && make build >/dev/null) || { printf '%sbuild failed%s\n' "$RED" "$OFF"; exit 1; }
  fi

  setup
  trap teardown EXIT

  section "cli"
  expect_cmd "exits 0 on clean quit" 0 "$BIN" --version
  expect_cmd_contains "--version reports a version" "osql" "$BIN" --version
  expect_cmd_contains "--help lists the flags" "--no-history" "$BIN" --help
  expect_cmd "unknown flag exits 1" 1 "$BIN" --frobnicate
  expect_cmd_contains "unknown flag explains itself" "osql --help" "$BIN" --frobnicate
  expect_cmd_contains "init reports the state directory" ".osql" "$BIN" init
  expect_cmd_contains "--help documents the dir flag" "--dir" "$BIN" --help
  expect_cmd "--root with no value exits 1" 1 "$BIN" --root
  expect_cmd "--root with init exits 1" 1 "$BIN" init --root /

  section "shell basics"
  expect_shell_contains "greeting advertises help and exit" 'exit
' 'type "help" for commands'
  expect_shell_contains "help lists builtins" 'help
exit
' "clear" "exit" "history" "files" "count(" "open" "new" "summary" "delete"
  expect_shell_contains "blank lines are ignored" '

files from '"'"'docs'"'"'
exit
' "notes.txt"
  expect_shell_contains "shell survives a bad command" 'filez from '"'"'docs'"'"'
files from '"'"'docs'"'"'
exit
' 'Did you mean "files"?' "notes.txt"

  section "select — targets"
  expect_names "all lists files and folders" "all from 'docs'" "Makefile notes.txt q4-report.txt report.pdf secret.txt"
  expect_names "files excludes folders" "files from 'src'" "helper.go test_lexer.go test_parser.go"
  expect_names "folders excludes files" "folders from 'src'" "big one"

  section "select — depth"
  expect_absent "non-recursive stays at one level" "files from 'nested'" "far.txt"
  expect_contains "recursive descends" "files from 'nested' recursive" "deep/far.txt"
  expect_names "recursive relative paths" "files from 'src' recursive where type = 'txt'" "big/f1.txt big/f10.txt big/f11.txt big/f2.txt big/f3.txt big/f4.txt big/f5.txt big/f6.txt big/f7.txt big/f8.txt big/f9.txt one/only.txt"

  section "select — where"
  expect_names "type filter" "files from 'docs' where type = 'txt'" "notes.txt q4-report.txt secret.txt"
  expect_names "type filter with dot is the same query" "files from 'docs' where type = '.txt'" "notes.txt q4-report.txt secret.txt"
  expect_names "name exact" "files from 'docs' where name = 'notes.txt'" "notes.txt"
  expect_names "name negated" "files from 'docs' where name != 'notes.txt'" "Makefile q4-report.txt report.pdf secret.txt"
  expect_names "name_like contains" "files from 'docs' where name_like = '%report%'" "q4-report.txt report.pdf"
  expect_names "name_like prefix" "files from 'docs' where name_like = 'q4%'" "q4-report.txt"
  expect_names "name_like suffix" "files from 'docs' where name_like = '%.pdf'" "report.pdf"
  expect_names "name_like with star alias" "files from 'docs' where name_like = '*report*'" "q4-report.txt report.pdf"
  expect_names "two predicates with and" "files from 'src' where name_like = 'test_%' and type = 'go'" "test_lexer.go test_parser.go"

  section "select — count(child)"
  expect_names "count greater than" "folders from 'src' where count(child) > 10" "big"
  expect_names "count equal" "folders from 'src' where count(child) = 1" "one"
  expect_names "count less or equal" "folders from 'src' where count(child) <= 1" "one"

  section "select — paths are root-relative"
  expect_names "bare path" "files from 'docs' where type = 'txt'" "notes.txt q4-report.txt secret.txt"
  expect_names "a bare folder name is relative" "files from 'docs' where type = 'txt'" "notes.txt q4-report.txt secret.txt"
  expect_names "tilde form" "files from '~/docs' where type = 'txt'" "notes.txt q4-report.txt secret.txt"
  expect_names "dot prefix form" "files from './docs' where type = 'txt'" "notes.txt q4-report.txt secret.txt"
  expect_contains "bare word without quotes" "files from docs" "notes.txt"
  expect_contains "dot is the root" "files from '.'" "app.log"
  expect_contains "tilde is the root" "folders from '~'" "docs"
  expect_contains "dot is the current folder" "folders from '.'" "docs"
  expect_contains "nested relative path" "files from 'nested/deep'" "far.txt"
  expect_contains "an absolute path is absolute" "files from '$FIXTURE/nested/deep'" "far.txt"
  mkdir -p "$WORK/sibling" && : > "$WORK/sibling/reachable.txt"
  expect_contains "paths above the start folder are reachable" "files from '../sibling'" "reachable.txt"
  expect_contains "absolute paths are reachable" "files from '$WORK/sibling'" "reachable.txt"
  expect_absent "a real system path is reachable" "folders from '/'" "I couldn't find a folder"

  section "select — lexing"
  expect_contains "uppercase keywords" "FILES FROM 'docs'" "notes.txt"
  expect_contains "trailing semicolon" "files from 'docs';" "notes.txt"
  expect_contains "operator without spaces" "files from 'docs' where type='txt'" "notes.txt"
  expect_contains "path with spaces stays one token" "files from 'docs' where name = 'q4-report.txt'" "q4-report.txt"

  section "count()"
  expect_line "count files" "count(files) from 'docs'" "files  5"
  expect_line "count folders" "count(folders) from 'src'" "folders  2"
  expect_names_from_count "count all splits into two rows" "count(all) from 'docs'" "files 5|folders 0"
  expect_names_from_count "count all with folders present" "count(all) from 'src'" "files 3|folders 2"
  expect_contains "count header" "count(files) from 'docs'" "WHAT" "COUNT"
  expect_line "count with a filter" "count(files) from 'docs' where type = 'txt'" "files  3"
  expect_line "count recursive" "count(files) from 'src' recursive" "files  15"
  expect_line "count zero matches" "count(files) from 'docs' where type = 'zzz'" "files  0"
  expect_line "count uppercase" "COUNT(FILES) FROM 'docs'" "files  5"
  expect_absent "count carries no row footer" "count(files) from 'docs'" " files" 
  expect_line "count singular is corrected" "count(file) from 'docs'" 'Use "files", not "file" — for example: files from '"'"'Documents'"'"''
  expect_line "count unknown target" "count(bogus) from 'docs'" 'I can list "files", "folders", "all", or "apps" — not "bogus".'
  expect_line "count unclosed paren" "count(files from 'docs'" "count( needs a closing ) — for example: count(files) from 'Documents'"
  expect_line "count empty parens" "count() from 'docs'" 'I need "files", "folders", or "all" to start — for example: files from '"'"'Documents'"'"''
  expect_line "count missing path" "count(files) from" 'I need a folder after "from" — for example: files from '"'"'Documents'"'"''

  section "output"
  expect_contains "four column header" "all from 'docs'" "NAME" "TYPE" "SIZE" "MODIFIED"
  expect_contains "folders show folder and no size" "folders from 'src'" "folder"
  expect_line "footer counts results" "files from 'docs' where name = 'notes.txt'" "1 files"
  expect_contains "extensionless file shows an em dash" "files from 'docs' where name = 'Makefile'" "—"
  expect_contains "bytes have no decimal" "files from '.' where name = 'size_1023.bin'" "1023 B"
  expect_contains "kilobytes at the boundary" "files from '.' where name = 'size_1024.bin'" "1.0 KB"
  expect_contains "1048575 promotes to megabytes" "files from '.' where name = 'size_1048575.bin'" "1.0 MB"
  expect_absent "never shows 1024 of a unit" "files from '.' where name = 'size_1048575.bin'" "1024.0 KB"

  section "open"
  expect_line "prints the first line" "open 'text/readme.txt'" "line one"
  expect_line "prints the second line" "open 'text/readme.txt'" "line two"
  expect_contains "reads a nested path" "open 'text/readme.txt'" "line one"
  expect_contains "tilde form works" "open '~/text/readme.txt'" "line one"
  expect_contains "bare word path works" "open text/readme.txt" "line one"
  expect_line "adds a missing final newline" "open 'text/bare.txt'" "no trailing newline"
  expect_line "keeps unicode intact" "open 'text/unicode.txt'" "日本語"
  expect_line "refuses a folder" "open 'text'" "'text' is a folder, not a file. Try: open 'text/notes.txt'"
  expect_line "refuses the root" "open '.'" "'.' is a folder, not a file. Try: open './notes.txt'"
  expect_line "missing file" "open 'nope.txt'" "I couldn't find a file at 'nope.txt'. Check the path and try again."
  expect_line "refuses binary" "open 'text/prog.bin'" "'text/prog.bin' looks like a binary file, so I won't print it. open only shows text."
  expect_absent "prints nothing for binary" "open 'text/prog.bin'" "ELF"
  expect_line "needs a path" "open" 'I need a file after "open" — for example: open '"'"'notes.txt'"'"''
  expect_contains "rejects trailing words" "open 'text/readme.txt' junk" 'I don'"'"'t understand "junk" here.'
  printf 'outside body\n' > "$WORK/sibling/outside.txt"
  expect_contains "open reads above the start folder" "open '../sibling/outside.txt'" "outside body"
  empty_out="$(osql "open 'text/empty.txt'")"
  if [ -z "$(printf '%s' "$empty_out" | tr -d '[:space:]')" ]; then
    pass "empty file prints nothing"
  else
    fail "empty file prints nothing" "expected no output" "$empty_out"
  fi

  section "new"
  expect_line "creates a file" "new file 'made.txt'" "Created 'made.txt'"
  expect_contains "the file is really there" "files from '.' where name = 'made.txt'" "made.txt"
  expect_line "creates a folder" "new folder 'made_dir'" "Created 'made_dir'"
  expect_contains "the folder is really there" "folders from '.' where name = 'made_dir'" "made_dir"
  expect_line "writes data" "new file 'greet.txt' data='hello hello line testing'" "Created 'greet.txt'"
  expect_line "data is readable back" "open 'greet.txt'" "hello hello line testing"
  expect_contains "creates missing parents" "new file 'deep/one/two/leaf.txt' data='way down here'" "Created 'deep/one/two/leaf.txt'" "also created:"
  expect_contains "reports which parents it made" "new file 'deep2/a/b.txt'" "deep2"
  expect_line "nested file is readable" "open 'deep/one/two/leaf.txt'" "way down here"
  expect_contains "existing parents are not reported" "new file 'deep/one/two/second.txt'" "Created 'deep/one/two/second.txt'"
  expect_absent "no also-created line when parents exist" "new file 'deep/one/two/third.txt'" "also created:"
  expect_line "refuses an existing file" "new file 'made.txt'" "'made.txt' already exists. Nothing was changed."
  expect_line "refuses an existing folder" "new folder 'made_dir'" "'made_dir' already exists. Nothing was changed."
  expect_line "data never overwrites" "new file 'greet.txt' data='destroyed'" "'greet.txt' already exists. Nothing was changed."
  expect_line "original content survives" "open 'greet.txt'" "hello hello line testing"
  expect_line "plural is corrected" "new files 'x.txt'" "Use \"file\", not \"files\" — you make one thing at a time: new file 'notes.txt'"
  expect_line "unknown kind" "new thing 'x'" 'I can make a "file" or a "folder" — not "thing".'
  expect_line "needs a kind" "new" 'I need "file" or "folder" after "new" — for example: new file '"'"'notes.txt'"'"''
  expect_line "needs a path" "new file" 'I need a path after "new file" — for example: new file '"'"'notes.txt'"'"''
  expect_line "data needs a value" "new file 'x.txt' data=" "data needs a value in quotes — for example: data='hello there'"
  expect_line "folders take no data" "new folder 'y' data='z'" "A folder can't hold data. Drop the data part, or use: new file 'notes.txt' data='hello'"
  osql "new file '../sibling/made.txt'" >/dev/null
  if [ -f "$WORK/sibling/made.txt" ]; then
    pass "new creates above the start folder"
  else
    fail "new creates above the start folder" "file was not created"
  fi
  if [ -e "$(dirname "$HOME")/escape.txt" ]; then
    fail "nothing is created outside the root" "escape.txt exists above HOME"
  else
    pass "nothing is created outside the root"
  fi

  section "escapes"
  expect_line "newline in data" "new file 'multi.txt' data='line one\\nline two'" "Created 'multi.txt'"
  expect_line "first line reads back" "open 'multi.txt'" "line one"
  expect_line "second line reads back" "open 'multi.txt'" "line two"
  expect_line "single quote in data" "new file 'apos.txt' data='it\\'s working'" "Created 'apos.txt'"
  expect_line "apostrophe reads back" "open 'apos.txt'" "it's working"
  expect_line "backslash in data" "new file 'bslash.txt' data='a\\\\b'" "Created 'bslash.txt'"
  expect_line "backslash reads back" "open 'bslash.txt'" 'a\b'
  expect_contains "escapes work in paths too" "files from 'docs' where name = 'notes.txt'" "notes.txt"
  expect_line "unknown escape is refused" "new file 'z.txt' data='a\\qb'" "I don't know the escape \"\\q\". I understand: \\n, \\t, \\r, \\\\ and \\'"
  expect_line "trailing backslash is unclosed" "files from 'abc\\" "This quote is never closed: 'abc\\ — add a closing '"

  section "summary"
  expect_contains "one level header" "summary from 'docs'" "docs — one level"
  expect_contains "recursive header" "summary from 'docs' recursive" "docs — all levels"
  expect_contains "counts section" "summary from 'docs'" "WHAT" "COUNT" "SIZE" "files" "folders" "total"
  expect_contains "types section" "summary from 'docs'" "TYPE"
  expect_contains "largest section" "summary from 'docs'" "LARGEST"
  expect_contains "modified range" "summary from 'docs'" "MODIFIED"
  expect_contains "long names are truncated" "summary from 'wide'" "…"
  expect_absent "long names do not stretch the table" "summary from 'wide'" "a-very-long-filename-that-should-be-truncated-in-the-summary-table.txt"
  expect_contains "wide characters survive" "summary from 'wide'" "日本語"
  wide_out="$(osql "summary from 'wide'")"
  widest=0
  while IFS= read -r line; do
    n=$(printf '%s' "$line" | wc -m | tr -d ' ')
    [ "$n" -gt "$widest" ] && widest=$n
  done <<< "$wide_out"
  if [ "$widest" -le 72 ]; then
    pass "summary fits a normal terminal ($widest columns)"
  else
    fail "summary fits a normal terminal" "widest line is $widest columns" "$wide_out"
  fi
  count_col_what="$(printf '%s' "$wide_out" | grep 'WHAT' | grep -bo 'COUNT' | cut -d: -f1)"
  count_col_type="$(printf '%s' "$wide_out" | grep 'TYPE' | grep -bo 'COUNT' | cut -d: -f1)"
  if [ "$count_col_what" = "$count_col_type" ]; then
    pass "count column lines up across blocks"
  else
    fail "count column lines up across blocks" "WHAT at $count_col_what, TYPE at $count_col_type" "$wide_out"
  fi
  expect_contains "folders carry no size" "summary from 'src'" "folders"
  expect_line "empty folder gets one line" "summary from 'truly_empty'" "'truly_empty' is empty."
  expect_contains "folder with no files" "summary from 'onlydirs'" "Contains 2 folders, and no files."
  expect_absent "no files means no largest table" "summary from 'onlydirs'" "LARGEST"
  expect_contains "warns about skipped folders" "summary from 'proj' recursive" "Skipped 2 folders" "node_modules" ".venv" 'Add "with skipped" to include them'
  expect_contains "warning mentions the time cost" "summary from 'proj' recursive" "take longer"
  expect_absent "with skipped drops the warning" "summary from 'proj' recursive with skipped" "Skipped 2 folders"
  expect_contains "with skipped counts more" "summary from 'proj' recursive with skipped" "js"
  expect_absent "default run excludes skipped types" "summary from 'proj' recursive" "6.0 KB"
  expect_line "with needs skipped" "summary from 'docs' with" '"with" needs "skipped" — for example: summary from '"'"'Documents'"'"' recursive with skipped'
  expect_line "with rejects other words" "summary from 'docs' with everything" 'I only know "with skipped", not "with everything".'
  expect_line "summary needs from" "summary 'docs'" 'I need "from" before the folder — for example: files from '"'"'Documents'"'"''
  expect_line "summary needs a path" "summary from" 'I need a folder after "from" — for example: files from '"'"'Documents'"'"''
  expect_line "missing folder" "summary from 'nope'" "I couldn't find a folder at 'nope'. Check the path and try again."
  expect_line "a file is not a folder" "summary from 'app.log'" "'app.log' is a file, not a folder. Try: files from 'Documents'"

  section "delete"
  mkdir -p "$FIXTURE/trashme" "$FIXTURE/keepme/inner"
  printf 'a' > "$FIXTURE/trashme/one.tmp"
  printf 'bb' > "$FIXTURE/trashme/two.tmp"
  printf 'ccc' > "$FIXTURE/trashme/keep.txt"
  printf 'd' > "$FIXTURE/keepme/inner/deep.txt"
  printf 'e' > "$FIXTURE/solo.txt"
  printf 'f' > "$FIXTURE/forever.txt"

  expect_script_contains "preview lists what will go" "delete files from 'trashme' where type = 'tmp'
no
exit
" "These 2 items will be deleted" "one.tmp" "two.tmp" "total" 'Type "yes" to go ahead'
  file_kept "cancelling keeps the files" "trashme/one.tmp"
  expect_script_contains "cancelling says so" "delete file 'solo.txt'
no
exit
" "Cancelled. Nothing was deleted."
  file_kept "a cancelled single delete keeps the file" "solo.txt"

  expect_script_contains "empty answer cancels" "delete file 'solo.txt'

exit
" "Cancelled."
  file_kept "an empty answer keeps the file" "solo.txt"

  expect_script_contains "end of input cancels" "delete file 'solo.txt'
" "Cancelled."
  file_kept "end of input keeps the file" "solo.txt"

  expect_script_contains "yes moves files to the trash" "delete files from 'trashme' where type = 'tmp'
yes
exit
" "Moved 2 items to the trash"
  file_gone "the tmp files are gone" "trashme/one.tmp"
  file_kept "unmatched files survive" "trashme/keep.txt"
  if [ -e "$HOME/.Trash/one.tmp" ] || [ -e "$HOME/.local/share/Trash/files/one.tmp" ]; then
    pass "the file reached the trash"
  else
    fail "the file reached the trash" "not found in the trash"
  fi

  expect_script_contains "permanently says there is no way back" "delete file 'forever.txt' permanently
no
exit
" "for good, with no way back"
  expect_script_contains "permanently deletes" "delete file 'forever.txt' permanently
yes
exit
" "Deleted 1 item."
  file_gone "the permanently deleted file is gone" "forever.txt"
  if [ -e "$HOME/.Trash/forever.txt" ]; then
    fail "a permanent delete skips the trash" "forever.txt is in the trash"
  else
    pass "a permanent delete skips the trash"
  fi

  expect_script_contains "a folder shows its weight" "delete folder 'keepme'
no
exit
" "'keepme' and everything in it" "1 file"
  file_kept "the folder survives a cancel" "keepme/inner/deep.txt"

  expect_contains "nothing matched" "delete files from 'trashme' where type = 'zzz'" "nothing to delete"
  expect_line "refuses to empty the filesystem root" "delete all from '/'" "I won't empty '/' itself. Name a folder inside it, or add a where clause."
  expect_line "refuses to empty home" "delete all from '~'" "I won't empty '~' itself. Name a folder inside it, or add a where clause."
  here_out="$(printf "cd docs\ndelete folder '.'\nexit\n" | (cd "$FIXTURE" && "$BIN") 2>&1)"
  if printf '%s' "$here_out" | grep -qF "is the folder you are in, so I won't delete it"; then
    pass "refuses to delete the folder you are in"
  else
    fail "refuses to delete the folder you are in" "no refusal" "$here_out"
  fi
  expect_line "wrong kind suggests the other" "delete file 'trashme'" "'trashme' is a folder, not a file. Try: delete folder 'trashme'"
  expect_line "wrong kind the other way" "delete folder 'trashme/keep.txt'" "'trashme/keep.txt' is a file, not a folder. Try: delete file 'trashme/keep.txt'"
  expect_line "missing file" "delete file 'nope.txt'" "I couldn't find a file at 'nope.txt'. Check the path and try again."
  : > "$WORK/sibling/doomed.txt"
  printf "delete file '../sibling/doomed.txt'\nyes\nexit\n" | (cd "$FIXTURE" && "$BIN") >/dev/null 2>&1
  if [ -f "$WORK/sibling/doomed.txt" ]; then
    fail "delete reaches above the start folder" "file survived"
  else
    pass "delete reaches above the start folder"
  fi
  expect_line "needs a target" "delete" 'I need "file", "folder", "files", "folders" or "all" after "delete" — for example: delete file '"'"'notes.txt'"'"''
  expect_line "unknown target" "delete thing 'x'" 'I can'"'"'t delete "thing". Try "delete file", "delete folder", or "delete files from".'
  expect_line "needs a path" "delete file" 'I need a path after "delete file" — for example: delete file '"'"'notes.txt'"'"''
  expect_line "bulk needs from" "delete files 'x'" 'I need "from" before the folder — for example: files from '"'"'Documents'"'"''

  section "outcomes"
  expect_line "no matches" "files from 'docs' where type = 'zzz'" "No files matched."
  expect_line "empty folder" "folders from 'empty_ish'" "'empty_ish' is empty."

  section "errors"
  expect_line "folder missing" "files from 'Documnets'" "I couldn't find a folder at 'Documnets'. Check the path and try again."
  expect_line "path is a file" "files from 'app.log'" "'app.log' is a file, not a folder. Try: files from 'Documents'"
  expect_line "unknown word suggests a target" "filez from 'docs'" 'I can list "files", "folders", "all", or "apps" — not "filez". Did you mean "files"?'
  expect_line "singular target" "file from 'docs'" 'Use "files", not "file" — for example: files from '"'"'Documents'"'"''
  expect_line "removed select verb is explained" "select files from 'docs'" 'Queries don'"'"'t need "select" — start with what you want: files from '"'"'Documents'"'"''
  expect_line "unknown target" "documents from 'docs'" 'I can list "files", "folders", "all", or "apps" — not "documents".'
  expect_line "missing target" "from 'docs'" 'I need "files", "folders", or "all" to start — for example: files from '"'"'Documents'"'"''
  expect_line "missing from" "files 'docs'" 'I need "from" before the folder — for example: files from '"'"'Documents'"'"''
  expect_line "missing path" "files from" 'I need a folder after "from" — for example: files from '"'"'Documents'"'"''
  expect_line "unknown field" "files from 'docs' where extension = 'txt'" 'I don'"'"'t know the field "extension". I understand: name, name_like, type'
  expect_line "unknown field lists count(child) for folders" "folders from 'docs' where extension = 'txt'" 'I don'"'"'t know the field "extension". I understand: name, name_like, type, count(child)'
  expect_contains "wrong operator for field" "files from 'docs' where name < 'b'" '"name" only works with = and !=.'
  expect_line "count(child) on files" "files from 'docs' where count(child) > 1" "count(child) describes folders, not files. Try: folders from 'Documents' where count(child) > 10"
  expect_line "count(child) needs a number" "folders from 'src' where count(child) > 'many'" "count(child) needs a number — for example: count(child) > 10"
  expect_line "unclosed quote" "files from 'docs" "This quote is never closed: 'docs — add a closing '"
  expect_contains "unexpected trailing input" "files from 'docs' junk" 'I don'"'"'t understand "junk" here.'
  expect_contains "query ends early" "files from 'docs' where" 'The query ends after "where".'
  expect_contains "or is not supported yet" "files from 'docs' where name = 'a' or name = 'b'" 'I don'"'"'t understand "or" here.'

  section "--dir override"
  dir_out="$(printf "files from '.'\nexit\n" | (cd "$FIXTURE" && "$BIN" --dir "$FIXTURE/docs") 2>&1)"
  if printf '%s' "$dir_out" | grep -qF "notes.txt"; then
    pass "--dir starts elsewhere"
  else
    fail "--dir starts elsewhere" "expected notes.txt from docs" "$dir_out"
  fi
  dir_eq="$(printf "pwd\nexit\n" | (cd "$FIXTURE" && "$BIN" --dir="$FIXTURE/docs") 2>&1)"
  if printf '%s' "$dir_eq" | grep -qF "$FIXTURE/docs"; then
    pass "--dir=path form works"
  else
    fail "--dir=path form works" "expected the docs path from pwd" "$dir_eq"
  fi
  root_alias="$(printf "pwd\nexit\n" | (cd "$FIXTURE" && "$BIN" --root "$FIXTURE/docs") 2>&1)"
  if printf '%s' "$root_alias" | grep -qF "$FIXTURE/docs"; then
    pass "--root still works as an alias"
  else
    fail "--root still works as an alias" "expected the docs path" "$root_alias"
  fi

  section "cd and pwd"
  cd_out="$(printf "pwd\ncd docs\npwd\ncd ..\npwd\nexit\n" | (cd "$FIXTURE" && "$BIN") 2>&1)"
  if printf '%s' "$cd_out" | grep -qF "$FIXTURE/docs"; then
    pass "cd moves into a folder"
  else
    fail "cd moves into a folder" "pwd never showed the docs path" "$cd_out"
  fi
  expect_contains "cd into a missing folder fails" "cd nowhere" "I couldn't find a folder at 'nowhere'"
  expect_contains "cd into a file fails" "cd docs/notes.txt" "that is a file, not a folder"
  expect_contains "pwd takes no folder" "pwd somewhere" 'takes no folder'
  cd_home="$(printf "cd\npwd\nexit\n" | (cd "$FIXTURE" && "$BIN") 2>&1)"
  if printf '%s' "$cd_home" | grep -qF "$HOME"; then
    pass "bare cd goes home"
  else
    fail "bare cd goes home" "expected \$HOME" "$cd_home"
  fi
  cd_back="$(printf "cd docs\ncd -\npwd\nexit\n" | (cd "$FIXTURE" && "$BIN") 2>&1)"
  if printf '%s' "$cd_back" | tail -2 | grep -qF "$FIXTURE"; then
    pass "cd - returns to the previous folder"
  else
    fail "cd - returns to the previous folder" "did not return" "$cd_back"
  fi
  prompt_out="$(printf "cd docs\nexit\n" | (cd "$FIXTURE" && "$BIN") 2>&1)"
  if printf '%s' "$prompt_out" | grep -q "osql ~/docs >"; then
    pass "the prompt shows where you are"
  else
    fail "the prompt shows where you are" "prompt did not show ~/docs" "$prompt_out"
  fi

  section "state"
  if [ -d "$HOME/.osql" ]; then
    pass "state directory created on startup"
    perms="$(file_mode "$HOME/.osql")"
    [ "$perms" = "700" ] && pass "state directory is 0700" || fail "state directory is 0700" "mode is $perms"
    hperms="$(file_mode "$HOME/.osql/history.txt")"
    [ "$hperms" = "600" ] && pass "history.txt is 0600" || fail "history.txt is 0600" "mode is $hperms"
    grep -q "^version:" "$HOME/.osql/system.txt" && pass "system.txt records a version" || fail "system.txt records a version" "no version line"
    grep -qF "files from 'docs'" "$HOME/.osql/history.txt" && pass "commands are recorded in history" || fail "commands are recorded in history" "query not found in history.txt"
  else
    fail "state directory created on startup" "$HOME/.osql does not exist"
  fi

  rm -rf "$HOME/.osql"
  printf "files from 'docs'\nexit\n" | (cd "$FIXTURE" && "$BIN" --no-history) >/dev/null 2>&1
  if [ -f "$HOME/.osql/history.txt" ]; then
    fail "--no-history writes no history file" "history.txt exists"
  else
    pass "--no-history writes no history file"
  fi

  section "apps"
  apps_out="$(osql "apps")"
  if printf '%s' "$apps_out" | grep -qE '^NAME +VERSION +SOURCE +MODIFIED$'; then
    pass "apps renders four columns"
    printf '%s' "$apps_out" | grep -qE '^[0-9]+ apps?$' \
      && pass "apps footer counts apps" \
      || fail "apps footer counts apps" "no '<n> apps' footer" "$apps_out"
  elif printf '%s' "$apps_out" | grep -qF "I didn't find any installed apps."; then
    pass "apps renders four columns (skipped: no apps on this host)"
    pass "apps footer counts apps (skipped: no apps on this host)"
  else
    fail "apps renders four columns" "neither a table nor the empty message" "$apps_out"
    fail "apps footer counts apps" "no output to count" "$apps_out"
  fi

  expect_contains "count(apps) answers with one row" "count(apps)" "WHAT" "apps"
  expect_absent "apps never shows a size column" "apps" "SIZE"

  expect_line "apps refuses a path" "apps from '/Applications'" \
    '"apps" already looks everywhere your system installs apps, so it needs no path. Try: apps'
  expect_line "apps refuses recursive" "apps recursive" \
    '"apps" is never recursive — looking inside an app would list the helpers it ships with as if they were apps. Try: apps'
  expect_contains "delete apps is refused" "delete apps from '/'" "I won't uninstall apps"
  expect_contains "apps rejects file-only fields" "apps where type = 'txt'" \
    '"type" doesn'"'"'t work with "apps"'
  expect_contains "files reject app-only fields" "files from 'docs' where version = '1'" \
    '"version" doesn'"'"'t work with "files"'
  expect_line "singular app suggests apps" "app" \
    'I can list "files", "folders", "all", or "apps" — not "app". Did you mean "apps"?'
  expect_contains "unmatched apps filter says so" "apps where name_like = '%zzzznope%'" "No apps matched."

  expect_absent "apps shows no size column by default" "apps" "SIZE"
  sized_out="$(osql "apps with size")"
  if printf '%s' "$sized_out" | grep -qE '^NAME +VERSION +SOURCE +SIZE +MODIFIED$'; then
    pass "apps with size adds a size column"
    printf '%s' "$sized_out" | grep -qE '^[0-9]+ apps?, .* on disk$' \
      && pass "apps with size totals the disk usage" \
      || fail "apps with size totals the disk usage" "no total in footer" "$sized_out"
  elif printf '%s' "$sized_out" | grep -qF "I didn't find any installed apps."; then
    pass "apps with size adds a size column (skipped: no apps on this host)"
    pass "apps with size totals the disk usage (skipped: no apps on this host)"
  else
    fail "apps with size adds a size column" "no size column" "$sized_out"
    fail "apps with size totals the disk usage" "no output" "$sized_out"
  fi

  expect_line "with needs size" "apps with" '"with" needs "size" — for example: apps with size'
  expect_line "with rejects other words" "apps with skipped" 'After "apps with" I only know "size", not "skipped".'
  expect_line "with size goes before where" "apps where source = 'macos' with size" \
    '"with size" goes before "where" — for example: apps with size where source = '"'"'homebrew'"'"''
  expect_contains "count(apps) refuses with size" "count(apps) with size" "A count has no size column"

  expect_apps_query "source alias hmb is accepted" "apps where source = 'hmb'"
  expect_absent "alias resolves to the canonical name" "apps where source = 'hmb'" "hmb "
  expect_apps_query "source alias brew is accepted" "apps where source = 'brew'"

  summary_out="$(osql "summary apps")"
  if printf '%s' "$summary_out" | grep -qF "Installed apps"; then
    pass "summary apps prints a heading"
    for block in WHAT SOURCE LARGEST MODIFIED; do
      printf '%s' "$summary_out" | grep -qE "^  $block" \
        && pass "summary apps has a $block block" \
        || fail "summary apps has a $block block" "block missing" "$summary_out"
    done
  elif printf '%s' "$summary_out" | grep -qF "I didn't find any installed apps."; then
    pass "summary apps prints a heading (skipped: no apps on this host)"
    for block in WHAT SOURCE LARGEST MODIFIED; do
      pass "summary apps has a $block block (skipped: no apps on this host)"
    done
  else
    fail "summary apps prints a heading" "unexpected output" "$summary_out"
  fi

  expect_line "summary apps refuses a path" "summary apps from 'docs'" \
    '"apps" already looks everywhere your system installs apps, so it needs no path. Try: apps'
  expect_line "summary apps refuses recursive" "summary apps recursive" \
    '"apps" is never recursive — looking inside an app would list the helpers it ships with as if they were apps. Try: apps'
  expect_contains "summary apps refuses where" "summary apps where source = 'hmb'" "A summary covers everything"
  expect_contains "folder summary still works" "summary from 'docs'" "WHAT" "COUNT"

  section "summary"
  printf '  %s%d passed%s' "$GREEN" "$PASS" "$OFF"
  if [ "$FAIL" -gt 0 ]; then
    printf ', %s%d failed%s\n' "$RED" "$FAIL" "$OFF"
    printf '\n  %sfailing:%s\n' "$BOLD" "$OFF"
    for f in "${FAILURES[@]}"; do printf '    %s✘%s %s\n' "$RED" "$OFF" "$f"; done
    printf '\n'
    exit 1
  fi
  printf '\n\n'
}

main "$@"
