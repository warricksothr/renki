---
schema: 4
id: TKT-01M6DCFST14NSS5XD1BV4ST039
title: "CI: run just ci automatically, require the plugin, real lint"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - tooling
assignees: []
milestone: null
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:42:34Z
updated_at: 2026-11-02T05:42:34Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-review
  name: ""
extensions: {}
---

## Description

`just ci` is meant to be the gate, but nothing enforces it and part of it is weaker than it looks.

### 1. No CI runs it

There is no `.github/` and no other runner. The root justfile says "There is no CI yet", while AGENTS.md says "CI runs it and nothing else". Fix: add a workflow that runs `just ci` on pull requests, on Linux with Go, .NET, Node with Playwright Chromium, and Python 3. Or, if CI is deliberately absent, correct AGENTS.md. Related: TKT-01M61V9F (Gate pushes on a local vuoro review).

### 2. The game-facing plugin build can be skipped and still pass

`plugin/justfile:20-22`: without `VALHEIM_DIR` or `game.local.props`, Renki.Plugin (about 9k lines of C#) is skipped with one line on stderr, and `just check` hides that line behind PASS. A compile error in the plugin first appears at `just package` or `vm deploy`.

Fix:
- Add `RENKI_REQUIRE_PLUGIN=1`, which makes the skip a failure, and set it wherever deploys happen.
- Make `scripts/check.sh` print any "skipped" lines beside PASS.
- Consider compiling against reference stubs so CI can build the plugin without the game.

### 3. "lint" only checks formatting

- Panel: `prettier --check` only, with no ESLint or type check over about 7k lines, including the 3k-line `controller.jsx`.
- Plugin: `dotnet format whitespace`, with `TreatWarningsAsErrors=false` and no analyzers.
- Go: gofmt and `go vet`, with no staticcheck.
- Python (about 6.6k lines) and PowerShell: not linted at all.

Fix:
- Add ESLint (`no-undef`, `no-unused-vars`) or `tsc --checkJs` to `just panel lint`.
- Add `staticcheck` to `just controller lint`.
- Turn on analyzers for Renki.Core.
- Add a pyflakes or ruff pass for `scripts/`.
- Fix whatever they find, or suppress it with a stated reason.

## Acceptance criteria

- [ ] just ci runs on every pull request, or AGENTS.md says why there is no CI
- [ ] A required-plugin mode fails when Renki.Plugin is skipped, and check.sh shows skips
- [ ] Lint runs ESLint or tsc for the panel, staticcheck for Go, analyzers for Renki.Core and pyflakes or ruff for scripts
