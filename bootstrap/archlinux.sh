#!/usr/bin/env bash
set -euo pipefail

DEFAULT_HOMEBASE_REPO="https://github.com/gin31259461/homebase.git"
DEFAULT_HOMEBASE_DIR="$HOME/.local/lib/homebase"
HOMEBASE_DIR_CUSTOM=0
if [[ -n "${HOMEBASE_DIR:-}" ]]; then
  HOMEBASE_DIR_CUSTOM=1
fi
HOMEBASE_DIR="${HOMEBASE_DIR:-$DEFAULT_HOMEBASE_DIR}"
HB_BIN="${HB_BIN:-$HOME/.local/bin/hb}"
HOMEBASE_REPO="$DEFAULT_HOMEBASE_REPO"

dir_is_nonempty() {
  local dir="$1"
  local entry
  while IFS= read -r -d '' entry; do
    return 0
  done < <(find "$dir" -mindepth 1 -maxdepth 1 -print0 2>/dev/null)
  return 1
}

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
    HOMEBASE_DIR_CUSTOM=1
    shift
    ;;
  *)
    args+=("$1")
    ;;
  esac
  shift
done

if [[ -z "${HOMEBASE_BOOTSTRAP_SKIP_PLATFORM_CHECK:-}" ]]; then
[[ -f /etc/arch-release || -f /etc/manjaro-release ]] || {
  printf 'ERROR: Homebase archlinux bootstrap targets Arch-family Linux only\n' >&2
  exit 1
}
fi

sudo pacman -S --needed --noconfirm git base-devel go

if [[ -d "$HOMEBASE_DIR/.git" ]]; then
  git -C "$HOMEBASE_DIR" pull --ff-only
elif [[ -f "$HOMEBASE_DIR/go.mod" ]]; then
  printf 'Using existing Homebase source at %s\n' "$HOMEBASE_DIR"
else
  if [[ "$HOMEBASE_DIR_CUSTOM" -eq 1 && -d "$HOMEBASE_DIR" ]] && dir_is_nonempty "$HOMEBASE_DIR"; then
    printf 'ERROR: refusing to replace non-empty custom HOMEBASE_DIR at %s\n' "$HOMEBASE_DIR" >&2
    exit 1
  fi
  rm -rf "$HOMEBASE_DIR"
  git clone --depth=1 "$HOMEBASE_REPO" "$HOMEBASE_DIR"
fi

export HOMEBASE_DIR
mkdir -p "$HOME/.local/bin"
go build -C "$HOMEBASE_DIR" -o "$HB_BIN" ./cmd/hb
exec "$HB_BIN" bootstrap "${args[@]}"
