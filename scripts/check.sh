#!/usr/bin/env bash
# Keep routine check output on disk, with bounded diagnostics in the terminal.
set -euo pipefail
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
if [ "$#" -eq 0 ] || [ "$1" = check ]; then
    echo 'usage: just check <recipe> [arguments...]' >&2
    exit 2
fi
mkdir -p "$root/.checks"
log=$(mktemp "$root/.checks/$(date -u +%Y%m%dT%H%M%SZ)-XXXXXX.log")
echo "Running just $*; full log: $log"
started=$SECONDS
# A pipe also protects the log from children opening /dev/stderr with
# truncation (for example tee in a formatting check).
if just --justfile "$root/justfile" "$@" 2>&1 | cat >"$log"; then
    echo "PASS: just $* ($((SECONDS - started))s); full log: $log"
else
    status=$?
    echo "FAIL: just $* (exit $status, $((SECONDS - started))s); last 80 lines:" >&2
    tail -n 80 "$log" >&2
    echo "Full log: $log" >&2
    exit "$status"
fi
