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
milestone: M4
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies:
  - TKT-01M6EFGJY8TTS36QZRACX5CW9X
  - TKT-01M6EFGJZKWV5XZTND3CH55E7W
  - TKT-01M6EFGK0R9CM18FEBJW68ZKWF
  - TKT-01M6EFGK2M7MBNZRRJABMBCMGM
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:42:34Z
updated_at: 2026-11-02T15:57:06Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
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

### Split into children

This ticket is the umbrella for goal 4.1. Each part is one pull request:

- TKT-01M6EFGJY8 (CI: run just ci on every pull request): item 1. Land it first; every other agent's pull request then gets the gate.
- TKT-01M6EFGJZK (CI: fail when Renki.Plugin is skipped, and show skips in check): item 2, including the owner decision on how CI compiles the plugin.
- TKT-01M6EFGK0R (Lint: staticcheck for Go, and ESLint or tsc for the panel): item 3, Go and panel.
- TKT-01M6EFGK2M (Lint: analyzers for Renki.Core, and ruff for scripts): item 3, C# and Python.

This ticket is done when all four are. Line numbers in the description refer to commit f1a5483; at 0b932ef the skip lives in `plugin/justfile`'s `has_game`, `skip_note` and `build`.

### Files

- `.github/workflows/` (new), `justfile`, `controller.just`, `panel/justfile`, `plugin/justfile`, `plugin/Directory.Build.props`, `scripts/check.sh`, `scripts/check-test.sh`, `AGENTS.md`

### Verification

- `just check ci` passes with every child landed.
- Goal 4.1's evidence: the workflow file, one green run, and one red run on a planted error, linked from TKT-01M6EFGJY8.

## Acceptance criteria

- [ ] just ci runs on every pull request, or AGENTS.md says why there is no CI
- [ ] A required-plugin mode fails when Renki.Plugin is skipped, and check.sh shows skips
- [ ] Lint runs ESLint or tsc for the panel, staticcheck for Go, analyzers for Renki.Core and pyflakes or ruff for scripts
