#!/usr/bin/env bash
set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  ./copy_jeppesen_from_wsl.sh <remote-user> <remote-repo-dir> [remote-host]

This script is intended to run under WSL on the Windows machine that has the
Jeppesen installation. It copies the required Jeppesen DLLs, fonts, font
config files, and live chart data to the Linux box over SSH.

Files are placed under:
  win_deps/lib/            DLLs baked into the image/runtime
  win_deps/fonts/          .jtf/.ttf fonts and font config files baked into the image/runtime
  win/data/Charts/         live chart data to mount/update separately

Optional environment overrides:
  JEPPVIEW_DIR   Default: /mnt/c/Program Files (x86)/Jeppesen/JeppView for Windows
  JEPP_COMMON_DIR Default: /mnt/c/ProgramData/Jeppesen/Common
  FONTS_DIR      Default: $JEPP_COMMON_DIR/Fonts
  CHARTS_DIR     Default: $JEPP_COMMON_DIR/TerminalCharts
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
fi

if [[ $# -lt 2 || $# -gt 3 ]]; then
    usage >&2
    exit 1
fi

REMOTE_USER="$1"
REMOTE_DIR="$2"
REMOTE_HOST="$3"
SSH_TARGET="${REMOTE_USER}@${REMOTE_HOST}"

JEPPVIEW_DIR="${JEPPVIEW_DIR:-/mnt/c/Program Files (x86)/Jeppesen/JeppView for Windows}"
JEPP_COMMON_DIR="${JEPP_COMMON_DIR:-/mnt/c/ProgramData/Jeppesen/Common}"
FONTS_DIR="${FONTS_DIR:-$JEPP_COMMON_DIR/Fonts}"
CHARTS_DIR="${CHARTS_DIR:-$JEPP_COMMON_DIR/TerminalCharts}"

require_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "Missing required command: $1" >&2
        exit 1
    fi
}

find_file_ci() {
    local dir="$1"
    local name="$2"

    find "$dir" -maxdepth 1 -type f -iname "$name" -print | head -n 1
}

copy_required() {
    local src_dir="$1"
    local name="$2"
    local dest="$3"
    local src

    src="$(find_file_ci "$src_dir" "$name")"
    if [[ -z "$src" ]]; then
        echo "Required file not found: $src_dir/$name" >&2
        exit 1
    fi

    cp -f "$src" "$dest"
}

copy_optional() {
    local src_dir="$1"
    local name="$2"
    local dest="$3"
    local src

    src="$(find_file_ci "$src_dir" "$name")"
    if [[ -n "$src" ]]; then
        cp -f "$src" "$dest"
        return 0
    fi

    return 1
}

quote_sq() {
    printf "%s" "$1" | sed "s/'/'\"'\"'/g"
}

require_cmd ssh
require_cmd tar
require_cmd find

if [[ ! -d "$JEPPVIEW_DIR" ]]; then
    echo "Jeppesen install directory not found: $JEPPVIEW_DIR" >&2
    exit 1
fi

if [[ ! -d "$FONTS_DIR" ]]; then
    echo "Font directory not found: $FONTS_DIR" >&2
    exit 1
fi

if [[ ! -d "$CHARTS_DIR" ]]; then
    echo "Chart metadata directory not found: $CHARTS_DIR" >&2
    exit 1
fi

STAGE_DIR="$(mktemp -d)"
trap 'rm -rf "$STAGE_DIR"' EXIT

mkdir -p \
    "$STAGE_DIR/win_deps/lib" \
    "$STAGE_DIR/win_deps/fonts" \
    "$STAGE_DIR/win/data/Charts"

copy_required "$JEPPVIEW_DIR" "mrvtcl.dll" "$STAGE_DIR/win_deps/lib/mrvtcl.dll"
copy_required "$JEPPVIEW_DIR" "mrvdrv.dll" "$STAGE_DIR/win_deps/lib/mrvdrv.dll"
copy_required "$JEPPVIEW_DIR" "zlib.dll" "$STAGE_DIR/win_deps/lib/zlib.dll"

copy_required "$FONTS_DIR" "jeppesen.tfl" "$STAGE_DIR/win_deps/fonts/jeppesen.tfl"
copy_required "$FONTS_DIR" "jeppesen.tls" "$STAGE_DIR/win_deps/fonts/jeppesen.tls"
copy_required "$FONTS_DIR" "lssdef.tcl" "$STAGE_DIR/win_deps/fonts/lssdef.tcl"

font_count=0
while IFS= read -r -d '' font_path; do
    cp -f "$font_path" "$STAGE_DIR/win_deps/fonts/$(basename "$font_path")"
    font_count=$((font_count + 1))
done < <(find "$FONTS_DIR" -maxdepth 1 -type f \( -iname '*.jtf' -o -iname '*.ttf' \) -print0 | sort -z)

if [[ "$font_count" -eq 0 ]]; then
    echo "No .jtf or .ttf fonts found in $FONTS_DIR" >&2
    exit 1
fi

copy_required "$CHARTS_DIR" "charts.bin" "$STAGE_DIR/win/data/Charts/charts.bin"
copy_required "$CHARTS_DIR" "charts.dbf" "$STAGE_DIR/win/data/Charts/charts.dbf"
copy_required "$CHARTS_DIR" "ctypes.dbf" "$STAGE_DIR/win/data/Charts/ctypes.dbf"
copy_required "$CHARTS_DIR" "Airports.dbf" "$STAGE_DIR/win/data/Charts/Airports.dbf"
copy_optional "$CHARTS_DIR" "vfrcharts.bin" "$STAGE_DIR/win/data/Charts/vfrcharts.bin" >/dev/null || true
copy_optional "$CHARTS_DIR" "vfrchrts.dbf" "$STAGE_DIR/win/data/Charts/vfrchrts.dbf" >/dev/null || true

REMOTE_DIR_Q="$(quote_sq "$REMOTE_DIR")"

echo "Copying Jeppesen runtime payload to $SSH_TARGET:$REMOTE_DIR"
ssh "$SSH_TARGET" "mkdir -p '$REMOTE_DIR_Q'"
tar -C "$STAGE_DIR" -cf - . | ssh "$SSH_TARGET" "tar -C '$REMOTE_DIR_Q' -xf -"

cat <<EOF
Done.

Copied into:
  $SSH_TARGET:$REMOTE_DIR

Placed in repo root:
  win_deps/
    lib/
      mrvtcl.dll
      mrvdrv.dll
      zlib.dll
    fonts/
      jeppesen.tfl
      jeppesen.tls
      lssdef.tcl
      all .jtf/.ttf font files
  win/data/Charts/
      charts.bin
      charts.dbf
      ctypes.dbf
      Airports.dbf
      vfrcharts.bin (if present)
      vfrchrts.dbf (if present)
EOF
