#!/usr/bin/env bash
# `just setup-worktree`: give a fresh worktree the ignored local files that
# name the VM and the game's assemblies, copied from another worktree of this
# repository. The main checkout is tried first, then the others, newest copy
# first. Files already here are kept. Only names are printed, never contents.
#
#   setup-worktree.sh ROOT [--check]    --check reports and copies nothing
set -euo pipefail

root="$1"
mode="${2:-copy}"
files=(vm.local plugin/game.local.props)

mapfile -t trees < <(git -C "$root" worktree list --porcelain | sed -n 's/^worktree //p')
status=0
for file in "${files[@]}"; do
    if [ -e "$root/$file" ]; then
        echo "have    $file"
        continue
    fi
    source=""
    newest=0
    for tree in "${trees[@]}"; do
        [ "$tree" -ef "$root" ] && continue
        [ -f "$tree/$file" ] || continue
        # The first listed worktree is the main checkout: it wins outright.
        if [ "$tree" = "${trees[0]}" ]; then source="$tree"; break; fi
        stamp="$(stat -c %Y "$tree/$file")"
        if [ "$stamp" -gt "$newest" ]; then newest="$stamp"; source="$tree"; fi
    done
    if [ -z "$source" ]; then
        echo "MISSING $file: no other worktree has it (docs/VM.md, \"Then, on the dev box\")" >&2
        status=1
    elif [ "$mode" = "--check" ]; then
        echo "would   $file  <- $source"
    else
        mkdir -p "$(dirname "$root/$file")"
        cp -p "$source/$file" "$root/$file"
        echo "copied  $file  <- $source"
    fi
done
exit "$status"
