# renki's task entry point. Run `just` to list the recipes.
#
# Every task a person, an agent or CI runs more than once is a recipe, and
# this file routes it. The verbs below (build, test, lint, fmt, ci, clean) run
# the same verb in every component that exists. Each component keeps its own
# recipes in a module, run in its own directory: `just controller build`,
# `just plugin build`. Tooling that spans the repository (review, tickets)
# lives here. AGENTS.md, "Tasks", has the rules.
#
# There is no CI yet. When there is, it runs `just ci`, so a green `just ci`
# here means a green run there.

set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

# The Go controller. Its recipes are in controller.just, beside go.mod.
mod controller 'controller.just'

# The C# BepInEx plugin. Optional until plugin/justfile exists.
mod? plugin

# The components the verbs route to, in order. Adding a component means a
# `mod` line above and its name here.
components := "controller plugin"

# List the recipes.
default:
    @just --list

# Build every component.
build: (route "build")

# Test every component.
test: (route "test")

# Lint every component.
lint: (route "lint")

# Format every component's sources.
fmt: (route "fmt")

# Remove every component's build output.
clean: (route "clean")

# Everything CI runs: each component's ci, then the ticket store.
ci: (route "ci") tickets-check

# Assemble dist/renki-vm/: everything the Windows VM needs, ready to copy over.
# The plugin goes in BepInEx/plugins/renki there; the rest in one folder.
package:
    #!/usr/bin/env bash
    set -euo pipefail
    out=dist/renki-vm
    rm -rf "$out" && mkdir -p "$out/plugin"
    just --justfile "{{justfile()}}" controller build-windows
    cp dist/controller.exe "$out/"
    (cd plugin && dotnet build Renki.Plugin -c Release --nologo -v quiet)
    cp plugin/Renki.Plugin/bin/Release/netstandard2.1/Renki.{Plugin,Core}.dll "$out/plugin/"
    cp scripts/vm/*.ps1 scripts/vm/bot.example.psd1 controller.example.json "$out/"
    cp docs/TESTING.md "$out/"
    echo "packaged $out ($(git describe --tags --always --dirty 2>/dev/null || echo devel)):"
    (cd "$out" && find . -type f | sort | sed 's/^/  /')

# Validate the ticket store, as CI will.
tickets-check:
    git ticket check --strict

# Takes a git range first, default origin/main...HEAD, and vuoro flags after
# it or instead of it: `just review -preset renki`, `just review main...HEAD
# -json`. Exit status is vuoro's: 0 clean, 1 findings, 3 failed. vuoro reviews
# commits only, so commit first.
[doc("Review this branch's commits with vuoro, before pushing.")]
review *ARGS:
    #!/usr/bin/env bash
    set -euo pipefail
    command -v vuoro >/dev/null || { echo "review: vuoro is not on PATH (github.com/terva-sh/vuoro)" >&2; exit 2; }
    args=({{ARGS}})
    if [ ${#args[@]} -eq 0 ] || [[ ${args[0]} == -* ]]; then
        args=("origin/main...HEAD" "${args[@]}")
    fi
    exec vuoro review -C "{{justfile_directory()}}" "${args[@]}"

# Run one verb in every component that exists. A component that is present
# but lacks the verb is an error rather than a skip, so a gap cannot hide
# inside a green `just ci`.
[private]
route verb:
    #!/usr/bin/env bash
    set -euo pipefail
    recipes=" $(just --justfile "{{justfile()}}" --summary) "
    for c in {{components}}; do
        if [[ $recipes != *" $c::"* ]]; then
            echo "-- $c: not present, skipped" >&2
            continue
        fi
        if [[ $recipes != *" $c::{{verb}} "* ]]; then
            echo "$c has no '{{verb}}' recipe; every component implements the shared verbs (AGENTS.md, Tasks)" >&2
            exit 1
        fi
        echo "-- $c {{verb}}" >&2
        just --justfile "{{justfile()}}" "$c" "{{verb}}"
    done
