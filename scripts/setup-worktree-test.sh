#!/usr/bin/env bash
# Offline check of setup-worktree.sh against throwaway repositories.
set -euo pipefail
script="$(cd "$(dirname "$0")" && pwd)/setup-worktree.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "setup-worktree-test: $1" >&2; exit 1; }

git init -q "$tmp/main"
git -C "$tmp/main" -c user.name=t -c user.email=t@example.invalid commit -q --allow-empty -m init
for name in old new fresh; do git -C "$tmp/main" worktree add -q "$tmp/$name" -b "$name"; done

# Nothing anywhere: every file is reported missing and nothing is created.
if out="$(bash "$script" "$tmp/fresh" 2>&1)"; then fail "passed with no source"; fi
grep -q 'MISSING vm.local' <<<"$out" || fail "did not name the missing file"

mkdir -p "$tmp/old/plugin" "$tmp/new/plugin"
echo old-target > "$tmp/old/vm.local"; echo old-props > "$tmp/old/plugin/game.local.props"
echo new-target > "$tmp/new/vm.local"; echo new-props > "$tmp/new/plugin/game.local.props"
touch -d '2026-01-01' "$tmp/old/vm.local" "$tmp/old/plugin/game.local.props"

# --check copies nothing and prints no contents.
out="$(bash "$script" "$tmp/fresh" --check)"
[ ! -e "$tmp/fresh/vm.local" ] || fail "--check copied a file"
grep -q 'target\|props$' <<<"$out" && fail "printed file contents"

# Anything but --check is refused rather than taken as a request to copy.
if bash "$script" "$tmp/fresh" --dry-run >/dev/null 2>&1; then fail "accepted an unknown argument"; fi
[ ! -e "$tmp/fresh/vm.local" ] || fail "an unknown argument copied a file"

# The newest sibling copy wins; an existing file is kept.
echo mine > "$tmp/fresh/vm.local"
bash "$script" "$tmp/fresh" >/dev/null
[ "$(cat "$tmp/fresh/vm.local")" = mine ] || fail "overwrote an existing file"
[ "$(cat "$tmp/fresh/plugin/game.local.props")" = new-props ] || fail "did not take the newest copy"

# The main checkout wins over any sibling.
rm "$tmp/fresh/vm.local"
echo main-target > "$tmp/main/vm.local"; touch -d '2025-01-01' "$tmp/main/vm.local"
bash "$script" "$tmp/fresh" >/dev/null
[ "$(cat "$tmp/fresh/vm.local")" = main-target ] || fail "did not prefer the main checkout"
echo "setup-worktree-test: ok"
