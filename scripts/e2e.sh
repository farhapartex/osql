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

  printf 'keep' > "$FIXTURE/empty_ish/.keep"
  printf 'far' > "$FIXTURE/nested/deep/far.txt"
  printf 'app' > "$FIXTURE/app.log"
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

osql_raw() {
  printf '%s\nexit\n' "$1" | (cd "$FIXTURE" && "$BIN") 2>&1
}

osql() {
  osql_raw "$1" | tail -n +2 | sed 's/^osql > //'
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
  expect_cmd_contains "--help documents the root flag" "--root" "$BIN" --help
  expect_cmd "--root with no value exits 1" 1 "$BIN" --root
  expect_cmd "--root with init exits 1" 1 "$BIN" init --root /

  section "shell basics"
  expect_shell_contains "greeting advertises help and exit" 'exit
' 'type "help" for commands'
  expect_shell_contains "help lists builtins" 'help
exit
' "clear" "exit" "history" "files"
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
  expect_names "leading slash means the root, not the filesystem" "files from '/docs' where type = 'txt'" "notes.txt q4-report.txt secret.txt"
  expect_names "tilde form" "files from '~/docs' where type = 'txt'" "notes.txt q4-report.txt secret.txt"
  expect_names "dot prefix form" "files from './docs' where type = 'txt'" "notes.txt q4-report.txt secret.txt"
  expect_contains "bare word without quotes" "files from docs" "notes.txt"
  expect_contains "dot is the root" "files from '.'" "app.log"
  expect_contains "tilde is the root" "folders from '~'" "docs"
  expect_contains "slash is the root" "folders from '/'" "docs"
  expect_contains "nested rooted path" "files from '/nested/deep'" "far.txt"
  expect_line "escaping the root is refused" "files from '../..'" "I can only look inside '$HOME'. '../..' points outside it."
  expect_line "system paths are not reachable" "files from '/etc'" "I couldn't find a folder at '/etc'. Check the path and try again."

  section "select — lexing"
  expect_contains "uppercase keywords" "FILES FROM 'docs'" "notes.txt"
  expect_contains "trailing semicolon" "files from 'docs';" "notes.txt"
  expect_contains "operator without spaces" "files from 'docs' where type='txt'" "notes.txt"
  expect_contains "path with spaces stays one token" "files from 'docs' where name = 'q4-report.txt'" "q4-report.txt"

  section "output"
  expect_contains "four column header" "all from 'docs'" "NAME" "TYPE" "SIZE" "MODIFIED"
  expect_contains "folders show folder and no size" "folders from 'src'" "folder"
  expect_line "footer counts results" "files from 'docs' where name = 'notes.txt'" "1 files"
  expect_contains "extensionless file shows an em dash" "files from 'docs' where name = 'Makefile'" "—"
  expect_contains "bytes have no decimal" "files from '.' where name = 'size_1023.bin'" "1023 B"
  expect_contains "kilobytes at the boundary" "files from '.' where name = 'size_1024.bin'" "1.0 KB"
  expect_contains "1048575 promotes to megabytes" "files from '.' where name = 'size_1048575.bin'" "1.0 MB"
  expect_absent "never shows 1024 of a unit" "files from '.' where name = 'size_1048575.bin'" "1024.0 KB"

  section "outcomes"
  expect_line "no matches" "files from 'docs' where type = 'zzz'" "No files matched."
  expect_line "empty folder" "folders from 'empty_ish'" "'empty_ish' is empty."

  section "errors"
  expect_line "folder missing" "files from 'Documnets'" "I couldn't find a folder at 'Documnets'. Check the path and try again."
  expect_line "path is a file" "files from 'app.log'" "'app.log' is a file, not a folder. Try: files from 'Documents'"
  expect_line "unknown word suggests a target" "filez from 'docs'" 'I can list "files", "folders", or "all" — not "filez". Did you mean "files"?'
  expect_line "singular target" "file from 'docs'" 'Use "files", not "file" — for example: files from '"'"'Documents'"'"''
  expect_line "removed select verb is explained" "select files from 'docs'" 'Queries don'"'"'t need "select" — start with what you want: files from '"'"'Documents'"'"''
  expect_line "unknown target" "documents from 'docs'" 'I can list "files", "folders", or "all" — not "documents".'
  expect_line "missing target" "from 'docs'" 'I need "files", "folders", or "all" to start — for example: files from '"'"'Documents'"'"''
  expect_line "missing from" "files 'docs'" 'I need "from" before the folder — for example: files from '"'"'Documents'"'"''
  expect_line "missing path" "files from" 'I need a folder after "from" — for example: files from '"'"'Documents'"'"''
  expect_line "unknown field" "files from 'docs' where extension = 'txt'" 'I don'"'"'t know the field "extension". I understand: name, name_like, type, count(child)'
  expect_contains "wrong operator for field" "files from 'docs' where name < 'b'" '"name" only works with = and !=.'
  expect_line "count(child) on files" "files from 'docs' where count(child) > 1" "count(child) describes folders, not files. Try: folders from 'Documents' where count(child) > 10"
  expect_line "count(child) needs a number" "folders from 'src' where count(child) > 'many'" "count(child) needs a number — for example: count(child) > 10"
  expect_line "unclosed quote" "files from 'docs" "This quote is never closed: 'docs — add a closing '"
  expect_contains "unexpected trailing input" "files from 'docs' junk" 'I don'"'"'t understand "junk" here.'
  expect_contains "query ends early" "files from 'docs' where" 'The query ends after "where".'
  expect_contains "or is not supported yet" "files from 'docs' where name = 'a' or name = 'b'" 'I don'"'"'t understand "or" here.'

  section "--root override"
  root_out="$(printf "folders from '/'\nexit\n" | (cd "$FIXTURE" && "$BIN" --root "$FIXTURE/docs") 2>&1)"
  if printf '%s' "$root_out" | grep -qF "is empty."; then
    pass "--root anchors elsewhere"
  else
    fail "--root anchors elsewhere" "expected docs to have no folders" "$root_out"
  fi
  root_eq="$(printf "files from '/'\nexit\n" | (cd "$FIXTURE" && "$BIN" --root="$FIXTURE/docs") 2>&1)"
  if printf '%s' "$root_eq" | grep -qF "notes.txt"; then
    pass "--root=path form works"
  else
    fail "--root=path form works" "expected notes.txt" "$root_eq"
  fi

  section "state"
  if [ -d "$HOME/.osql" ]; then
    pass "state directory created on startup"
    perms="$(stat -f '%Lp' "$HOME/.osql" 2>/dev/null || stat -c '%a' "$HOME/.osql")"
    [ "$perms" = "700" ] && pass "state directory is 0700" || fail "state directory is 0700" "mode is $perms"
    hperms="$(stat -f '%Lp' "$HOME/.osql/history.txt" 2>/dev/null || stat -c '%a' "$HOME/.osql/history.txt")"
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
