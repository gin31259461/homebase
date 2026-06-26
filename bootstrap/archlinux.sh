#!/usr/bin/env bash
set -euo pipefail

DEFAULT_HOMEBASE_REPO="https://github.com/gin31259461/homebase.git"
HOMEBASE_DIR="${HOMEBASE_DIR:-$HOME/.local/lib/homebase}"
HB_BIN="${HB_BIN:-$HOME/.local/bin/hb}"
HOMEBASE_REPO="$DEFAULT_HOMEBASE_REPO"

args=()
while [[ $# -gt 0 ]]; do
  case "$1" in
  --homebase-repo)
    [[ -n "${2:-}" ]] || {
      printf 'ERROR: --homebase-repo requires a value\n' >&2
      exit 1
    }
    HOMEBASE_REPO="$2"
    shift
    ;;
  --homebase-dir)
    [[ -n "${2:-}" ]] || {
      printf 'ERROR: --homebase-dir requires a value\n' >&2
      exit 1
    }
    HOMEBASE_DIR="$2"
    shift
    ;;
  *)
    args+=("$1")
    ;;
  esac
  shift
done

[[ -f /etc/arch-release || -f /etc/manjaro-release ]] || {
  printf 'ERROR: Homebase archlinux bootstrap targets Arch-family Linux only\n' >&2
  exit 1
}

sudo pacman -S --needed --noconfirm git base-devel go rsync ca-certificates zsh

if [[ -d "$HOMEBASE_DIR/.git" ]]; then
  git -C "$HOMEBASE_DIR" pull --ff-only
elif [[ -f "$HOMEBASE_DIR/go.mod" ]]; then
  printf 'Using existing Homebase source at %s\n' "$HOMEBASE_DIR"
else
  rm -rf "$HOMEBASE_DIR"
  git clone --depth=1 "$HOMEBASE_REPO" "$HOMEBASE_DIR"
fi

mkdir -p "$HOME/.local/bin"
go build -C "$HOMEBASE_DIR" -o "$HB_BIN" ./cmd/hb
exec "$HB_BIN" bootstrap "${args[@]}"
exec "$HB_BIN" install --group cli-tools --yes
