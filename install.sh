#!/bin/sh
set -eu

REPO="farhapartex/osql"
BINARY="osql"
DEFAULT_DIR="$HOME/.local/bin"

INSTALL_DIR="${OSQL_INSTALL_DIR:-$DEFAULT_DIR}"
VERSION="${OSQL_VERSION:-}"
KEEP_DOWNLOAD=""

usage() {
  cat <<EOF
Install $BINARY.

  install.sh [--dir <path>] [--version <tag>]

  --dir <path>       where to put the binary (default: $DEFAULT_DIR)
  --version <tag>    install a specific release, e.g. v0.1.0
  --help             show this message

Environment variables work too: OSQL_INSTALL_DIR, OSQL_VERSION.
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --dir) shift; [ $# -gt 0 ] || { echo "--dir needs a path" >&2; exit 1; }; INSTALL_DIR="$1" ;;
    --dir=*) INSTALL_DIR="${1#--dir=}" ;;
    --version) shift; [ $# -gt 0 ] || { echo "--version needs a tag" >&2; exit 1; }; VERSION="$1" ;;
    --version=*) VERSION="${1#--version=}" ;;
    --help|-h) usage; exit 0 ;;
    *) echo "I don't understand \"$1\". Try --help." >&2; exit 1 ;;
  esac
  shift
done

say() { printf '%s\n' "$*"; }
die() { printf '%s\n' "$*" >&2; exit 1; }

need() {
  command -v "$1" >/dev/null 2>&1
}

if need curl; then
  fetch() { curl -fsSL "$1" -o "$2"; }
  fetch_stdout() { curl -fsSL "$1"; }
elif need wget; then
  fetch() { wget -qO "$2" "$1"; }
  fetch_stdout() { wget -qO- "$1"; }
else
  die "I need curl or wget to download anything, and found neither."
fi

if need sha256sum; then
  checksum() { sha256sum "$1" | cut -d' ' -f1; }
elif need shasum; then
  checksum() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
  die "I need sha256sum or shasum to check the download, and found neither."
fi

detect_platform() {
  os="$(uname -s)"
  arch="$(uname -m)"

  case "$os" in
    Darwin) os="darwin" ;;
    Linux) os="linux" ;;
    *) die "$BINARY runs on macOS and Linux. This looks like \"$os\"." ;;
  esac

  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *) die "I don't have a build for \"$arch\". Builds exist for amd64 and arm64." ;;
  esac

  printf '%s_%s' "$os" "$arch"
}

latest_version() {
  fetch_stdout "https://api.github.com/repos/$REPO/releases?per_page=1" |
    tr ',' '\n' |
    grep '"tag_name"' |
    head -n 1 |
    sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/'
}

platform="$(detect_platform)"

if [ -z "$VERSION" ]; then
  say "Looking for the newest release..."
  VERSION="$(latest_version)"
  [ -n "$VERSION" ] || die "I couldn't work out the latest version. Try again with --version v0.1.0."
fi

archive="${BINARY}_${VERSION}_${platform}.tar.gz"
base="https://github.com/$REPO/releases/download/$VERSION"

workdir="$(mktemp -d)"
if [ -z "$KEEP_DOWNLOAD" ]; then
  trap 'rm -rf "$workdir"' EXIT INT TERM
fi

say "Downloading $BINARY $VERSION for $platform..."
fetch "$base/$archive" "$workdir/$archive" ||
  die "I couldn't download $archive.
Check that $VERSION exists: https://github.com/$REPO/releases"

fetch "$base/checksums.txt" "$workdir/checksums.txt" ||
  die "I downloaded $archive but not its checksums, so I won't install it."

expected="$(grep " \{1,2\}$archive\$" "$workdir/checksums.txt" | cut -d' ' -f1 || true)"
[ -n "$expected" ] || die "$archive is not listed in checksums.txt, so I can't verify it."

actual="$(checksum "$workdir/$archive")"
if [ "$expected" != "$actual" ]; then
  die "The download does not match its checksum, so I stopped.
  expected $expected
  got      $actual"
fi
say "Checksum verified."

tar -xzf "$workdir/$archive" -C "$workdir" ||
  die "I couldn't unpack $archive."

unpacked="$workdir/${BINARY}_${VERSION}_${platform}/$BINARY"
[ -f "$unpacked" ] || unpacked="$(find "$workdir" -type f -name "$BINARY" -perm -u+x | head -n 1)"
[ -n "$unpacked" ] && [ -f "$unpacked" ] || die "I unpacked the archive but found no $BINARY inside it."

mkdir -p "$INSTALL_DIR" || die "I couldn't create $INSTALL_DIR."
if [ ! -w "$INSTALL_DIR" ]; then
  die "I can't write to $INSTALL_DIR.
Pick somewhere you own with --dir, for example: --dir \"\$HOME/.local/bin\""
fi

install -m 0755 "$unpacked" "$INSTALL_DIR/$BINARY" 2>/dev/null ||
  { cp "$unpacked" "$INSTALL_DIR/$BINARY" && chmod 0755 "$INSTALL_DIR/$BINARY"; } ||
  die "I couldn't put $BINARY into $INSTALL_DIR."

say ""
say "Installed $("$INSTALL_DIR/$BINARY" --version) to $INSTALL_DIR/$BINARY"

case ":$PATH:" in
  *":$INSTALL_DIR:"*)
    say ""
    say "Run it with:  $BINARY"
    ;;
  *)
    say ""
    say "$INSTALL_DIR is not on your PATH yet. Add this to your shell config:"
    say ""
    say "    export PATH=\"\$PATH:$INSTALL_DIR\""
    say ""
    say "Until then, run it with:  $INSTALL_DIR/$BINARY"
    ;;
esac
