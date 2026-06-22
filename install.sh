#!/usr/bin/env sh
# devwork installer (curl | bash fallback for non-Homebrew systems).
#
#   curl -fsSL https://raw.githubusercontent.com/iampapagray/devwork/main/install.sh | sh
#
# Honors:
#   DEVWORK_VERSION   pin a release tag (default: latest)
#   PREFIX            install dir (default: ~/.local/bin, or /usr/local/bin)
set -eu

REPO="iampapagray/devwork"
BINARY="devwork"

err() { printf 'error: %s\n' "$*" >&2; exit 1; }
info() { printf '%s\n' "$*" >&2; }

need() { command -v "$1" >/dev/null 2>&1 || err "missing required tool: $1"; }
need uname
need tar

# Prefer curl, fall back to wget.
if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1"; }
  fetch_out() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO- "$1"; }
  fetch_out() { wget -qO "$2" "$1"; }
else
  err "need curl or wget"
fi

# ── detect OS / arch ──────────────────────────────────────────────────────
os="$(uname -s)"
case "$os" in
  Linux)  OS=linux ;;
  Darwin) OS=darwin ;;
  *) err "unsupported OS: $os (Windows: download the zip from Releases)" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) err "unsupported arch: $arch" ;;
esac

# ── resolve version ───────────────────────────────────────────────────────
VERSION="${DEVWORK_VERSION:-}"
if [ -z "$VERSION" ]; then
  VERSION="$(fetch "https://api.github.com/repos/$REPO/releases/latest" \
    | grep '"tag_name"' | head -n1 | cut -d'"' -f4)"
  [ -n "$VERSION" ] || err "could not resolve the latest release"
fi
NUM="${VERSION#v}"

# ── download + verify ─────────────────────────────────────────────────────
ARCHIVE="${BINARY}_${NUM}_${OS}_${ARCH}.tar.gz"
BASE="https://github.com/$REPO/releases/download/$VERSION"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

info "Downloading $ARCHIVE ($VERSION)…"
fetch_out "$BASE/$ARCHIVE" "$TMP/$ARCHIVE" || err "download failed: $BASE/$ARCHIVE"

if fetch_out "$BASE/checksums.txt" "$TMP/checksums.txt" 2>/dev/null; then
  if command -v sha256sum >/dev/null 2>&1; then
    sumcmd="sha256sum"
  elif command -v shasum >/dev/null 2>&1; then
    sumcmd="shasum -a 256"
  else
    sumcmd=""
  fi
  if [ -n "$sumcmd" ]; then
    want="$(grep " $ARCHIVE\$" "$TMP/checksums.txt" | awk '{print $1}')"
    got="$( $sumcmd "$TMP/$ARCHIVE" | awk '{print $1}')"
    [ "$want" = "$got" ] || err "checksum mismatch for $ARCHIVE"
    info "Checksum verified."
  fi
else
  info "warning: checksums.txt not found; skipping verification."
fi

tar -xzf "$TMP/$ARCHIVE" -C "$TMP"

# ── install ───────────────────────────────────────────────────────────────
if [ -n "${PREFIX:-}" ]; then
  DEST="$PREFIX"
elif [ -w /usr/local/bin ] 2>/dev/null; then
  DEST="/usr/local/bin"
else
  DEST="$HOME/.local/bin"
fi
mkdir -p "$DEST"

install -m 0755 "$TMP/$BINARY" "$DEST/$BINARY"
ln -sf "$DEST/$BINARY" "$DEST/devw"
info "Installed $BINARY and devw to $DEST"

case ":$PATH:" in
  *":$DEST:"*) ;;
  *) info "note: add $DEST to your PATH:  export PATH=\"$DEST:\$PATH\"" ;;
esac

"$DEST/$BINARY" --version >&2 || true
