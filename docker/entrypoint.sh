#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="/app"
STATE_ROOT="${STATE_ROOT:-/var/lib/marinvent}"
CHARTS_ROOT="${CHARTS_ROOT:-$APP_ROOT/data/Charts}"
TCL_CACHE_DIR="${TCL_CACHE_DIR:-$CHARTS_ROOT/TCLs}"
CHARTS_BIN="${CHARTS_BIN:-$CHARTS_ROOT/charts.bin}"

export HOST="${HOST:-0.0.0.0}"
export PORT="${PORT:-8080}"
export WINEARCH="${WINEARCH:-win32}"
export WINEPREFIX="${WINEPREFIX:-$STATE_ROOT/wine}"
export MARINVENT_WINE_XVFB="${MARINVENT_WINE_XVFB:-1}"
export MARINVENT_PDF_PRINTER="${MARINVENT_PDF_PRINTER:-PDF}"

log() {
    printf '[entrypoint] %s\n' "$*"
}

require_file() {
    local path="$1"
    if [[ ! -f "$path" ]]; then
        log "missing required file: $path"
        exit 1
    fi
}

find_file_ci() {
    local dir="$1"
    local name="$2"

    find "$dir" -maxdepth 1 -type f -iname "$name" -print | head -n 1
}

ensure_runtime_dirs() {
    mkdir -p "$STATE_ROOT" "$CHARTS_ROOT" /run/cups/certs
}

configure_env() {
    export CHARTS_BIN="${CHARTS_BIN:-$(find_file_ci "$CHARTS_ROOT" "charts.bin")}"
    export CHARTS_DBF="${CHARTS_DBF:-$(find_file_ci "$CHARTS_ROOT" "charts.dbf")}"
    export TYPES_DBF="${TYPES_DBF:-$(find_file_ci "$CHARTS_ROOT" "ctypes.dbf")}"
    export AIRPORTS_DBF="${AIRPORTS_DBF:-$(find_file_ci "$CHARTS_ROOT" "Airports.dbf")}"
    export TCL_DIR="${TCL_DIR:-$TCL_CACHE_DIR}"

    if [[ -f "${VFR_CHARTS_DBF:-}" ]]; then
        export VFR_CHARTS_DBF
    elif [[ -n "$(find_file_ci "$CHARTS_ROOT" "vfrchrts.dbf")" ]]; then
        export VFR_CHARTS_DBF="$(find_file_ci "$CHARTS_ROOT" "vfrchrts.dbf")"
    else
        export VFR_CHARTS_DBF=""
    fi
}

validate_mount_contents() {
    require_file "$CHARTS_BIN"
    require_file "$CHARTS_DBF"
    require_file "$TYPES_DBF"
    require_file "$AIRPORTS_DBF"
}

ensure_cups() {
    if lpstat -p PDF >/dev/null 2>&1; then
        return
    fi

    log "starting CUPS"
    cupsd

    for _ in $(seq 1 30); do
        if lpstat -p PDF >/dev/null 2>&1; then
            return
        fi

        if lpinfo -v 2>/dev/null | grep -q 'cups-pdf:/'; then
            lpadmin -p PDF -E -v cups-pdf:/ -P /usr/share/ppd/cups-pdf/CUPS-PDF_noopt.ppd >/dev/null 2>&1 || true
            cupsenable PDF >/dev/null 2>&1 || true
            cupsaccept PDF >/dev/null 2>&1 || true
            if lpstat -p PDF >/dev/null 2>&1; then
                return
            fi
        fi

        sleep 1
    done

    log "CUPS PDF printer did not become ready"
    lpstat -p || true
    exit 1
}

ensure_wine_prefix() {
    if [[ -d "$WINEPREFIX/drive_c" ]]; then
        return
    fi

    log "initializing Wine prefix at $WINEPREFIX"
    if [[ "$MARINVENT_WINE_XVFB" == "1" ]] && command -v xvfb-run >/dev/null 2>&1; then
        WINEDEBUG=-all xvfb-run -a wineboot -u >/tmp/marinvent-wineboot.log 2>&1
    else
        WINEDEBUG=-all wineboot -u >/tmp/marinvent-wineboot.log 2>&1
    fi
}

sync_tcls() {
    local checksum_file="$CHARTS_ROOT/.charts.bin.sha256"
    local current_checksum
    current_checksum="$(sha256sum "$CHARTS_BIN" | awk '{print $1}')"

    if [[ -f "$checksum_file" ]] && [[ -d "$TCL_CACHE_DIR" ]] && [[ "$(cat "$checksum_file")" == "$current_checksum" ]]; then
        log "reusing extracted TCL cache"
        return
    fi

    local tmp_dir="$CHARTS_ROOT/.TCLs.tmp.$$"
    rm -rf "$tmp_dir"
    mkdir -p "$tmp_dir"

    log "extracting TCL files from charts.bin"
    (
        cd "$tmp_dir"
        python3 "$APP_ROOT/jdmtool/chartview.py" -x "$CHARTS_BIN" >"$CHARTS_ROOT/.chart-extract.log"
    )

    rm -rf "$TCL_CACHE_DIR"
    mv "$tmp_dir" "$TCL_CACHE_DIR"
    printf '%s\n' "$current_checksum" >"$checksum_file"
}

start_server() {
    log "starting API on ${HOST}:${PORT}"
    exec "$APP_ROOT/marinvent-api" \
        -host "$HOST" \
        -port "$PORT" \
        -charts "$CHARTS_DBF" \
        -vfrcharts "$VFR_CHARTS_DBF" \
        -types "$TYPES_DBF" \
        -airports "$AIRPORTS_DBF" \
        -tcls "$TCL_DIR"
}

ensure_runtime_dirs
configure_env
validate_mount_contents
ensure_cups
ensure_wine_prefix
sync_tcls
start_server
