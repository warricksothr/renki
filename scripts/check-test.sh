#!/usr/bin/env bash
set -euo pipefail
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
fixture=$(mktemp -d)
logs=()
trap 'rm -rf "$fixture"; for log in "${logs[@]}"; do rm -f "$log"; done' EXIT
cat >"$fixture/just" <<'SH'
#!/usr/bin/env bash
# Model just's --quiet option: it suppresses child output too.
for arg in "$@"; do
    if [ "$arg" = --quiet ]; then exit "${CHECK_TEST_EXIT:-0}"; fi
done
echo 'first diagnostic'
echo 'stderr via tee' | tee /dev/stderr >/dev/null
for ((i=0; i<100; i++)); do echo "detail $i"; done
echo 'last diagnostic' >&2
exit "${CHECK_TEST_EXIT:-0}"
SH
chmod +x "$fixture/just"
for expected in 0 37; do
    status=0
    PATH="$fixture:$PATH" CHECK_TEST_EXIT="$expected" \
        bash "$root/scripts/check.sh" ci >"$fixture/output" 2>&1 || status=$?
    [ "$status" -eq "$expected" ]
    IFS= read -r first <"$fixture/output"
    log=${first##*full log: }
    logs+=("$log")
    # Both streams must remain in the full log, even on success.
    rg -q '^first diagnostic$' "$log"
    rg -q '^stderr via tee$' "$log"
    rg -q '^last diagnostic$' "$log"
    if [ "$expected" -eq 0 ]; then
        rg -q '^PASS:' "$fixture/output"
        [ "$(wc -l <"$fixture/output")" -eq 2 ]
    else
        rg -q '^FAIL:.*exit 37' "$fixture/output"
        rg -q '^last diagnostic$' "$fixture/output"
        ! rg -q '^first diagnostic$' "$fixture/output"
        [ "$(wc -l <"$fixture/output")" -eq 83 ]
    fi
done
[ "${logs[0]}" != "${logs[1]}" ]
echo 'check wrapper: success, failure, full logs and bounded diagnostics passed'
