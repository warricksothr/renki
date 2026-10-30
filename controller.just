# The Go controller's recipes, loaded by the root justfile as the
# `controller` module: `just controller build`. They run in the repository
# root, where go.mod is. The root verbs route here; see AGENTS.md, "Tasks".

set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

# `-buildvcs=false` inside a linked git worktree, and empty everywhere else.
#
# A linked worktree has a `.git` file rather than a directory, so Go cannot
# stamp VCS information and linking a binary fails with "error obtaining VCS
# status". `go vet` and `go test` are unaffected. Agents work in worktrees, so
# without this every `just build` they run fails. The two git paths agree in
# the main tree and differ in a linked worktree.
buildvcs := `if [ "$(git rev-parse --git-dir 2>/dev/null)" != "$(git rev-parse --git-common-dir 2>/dev/null)" ]; then echo "-buildvcs=false"; fi`

version := `git describe --tags --always --dirty 2>/dev/null || echo devel`
ldflags := "-s -w -X main.version=" + version

# List the controller's recipes.
default:
    @just --list controller

# Build the controller into the repository root (gitignored).
build:
    go build {{buildvcs}} -trimpath -ldflags "{{ldflags}}" -o controller ./cmd/controller
    @echo "built ./controller ({{version}})"

# Cross-compile the controller for the Windows VM into dist/.
build-windows:
    GOOS=windows GOARCH=amd64 go build {{buildvcs}} -trimpath -ldflags "{{ldflags}}" -o dist/controller.exe ./cmd/controller
    @echo "built dist/controller.exe ({{version}})"

# Run the controller, with controller.local.json as its config when present.
run *ARGS:
    #!/usr/bin/env bash
    set -euo pipefail
    cfg=()
    [ -f controller.local.json ] && cfg=(-config controller.local.json)
    exec go run {{buildvcs}} ./cmd/controller "${cfg[@]}" {{ARGS}}

# Talk to a running controller with no game: stdin lines become chat.
fake *ARGS:
    go run {{buildvcs}} ./cmd/fakeplugin {{ARGS}}

# Takes files or directories (default ./audit) and flags: `just controller
# audit -task t-6`, `-last 10m`, `-since 16:40 -state 1s`, `-json`.
[doc("Print the audit log as a timeline (internal/audit).")]
audit *ARGS:
    go run {{buildvcs}} ./cmd/audit {{ARGS}}

# Run the tests.
test *ARGS:
    go test ./... {{ARGS}}

# Run only the named packages during development; test/ci still cover all.
[positional-arguments]
test-packages +PACKAGES:
    go test "$@"

# Run the tests under the race detector.
test-race *ARGS:
    go test -race ./... {{ARGS}}

# Terrain interpolation/shoreline checks use Node's built-in test runner.
test-map:
    node --test internal/panel/terrain_test.cjs

# Read-only real-assets preview; pass --serve, or captured --status/--map JSON.
[positional-arguments]
preview-map *ARGS:
    python3 scripts/map-preview.py "$@"

# Run go vet.
vet:
    go vet ./...

# Check formatting, then vet.
lint:
    @test -z "$(gofmt -l . | tee /dev/stderr)" || { echo "gofmt issues (run \`just fmt\`)" >&2; exit 1; }
    node --check internal/panel/static/app.js
    go vet ./...

# Format all Go sources.
fmt:
    gofmt -w .

# Tidy go.mod and go.sum.
tidy:
    go mod tidy

# Everything CI runs for the controller.
ci: lint test-race test-map build build-windows

# Remove build output.
clean:
    rm -f ./controller
    rm -rf ./dist
