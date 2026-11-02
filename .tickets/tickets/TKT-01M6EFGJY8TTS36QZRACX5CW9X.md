---
schema: 4
id: TKT-01M6EFGJY8TTS36QZRACX5CW9X
title: "CI: run just ci on every pull request"
type: task
status: ready
status_reason: null
priority: high
due_on: null
labels:
  - tooling
assignees: []
milestone: M4
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:54:40Z
updated_at: 2026-11-02T19:17:51Z
created_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
updated_by:
  id: agent:claude-code/renki-wave1
  name: ""
extensions: {}
---

## Description

Goal 4.1 in docs/ROADMAP.md says `just ci` runs on every pull request. Nothing runs it today: there is no `.github/` directory, and the root `justfile` still says "There is no CI yet". Every other M4 agent relies on this gate, so this is the first M4 ticket to land. Split out of TKT-01M6DCFST1 (CI: run just ci automatically, require the plugin, real lint), item 1.

### What to build

- A workflow that runs `just ci` and nothing else on every pull request and on pushes to main, on Linux.
- Toolchains it needs, read from the repository rather than guessed: Go from `go.mod` (`go 1.27.0`), the .NET 8 SDK (`plugin/Renki.Tests/Renki.Tests.csproj` targets `net8.0`), Node 20 or newer (`panel/package.json` engines), Playwright Chromium through `just panel ui-install`, Python 3 for the `vm::test-*`, `broadcast-test` and `streaming::test` recipes, `just`, and the `git-ticket` binary for `just tickets-check`.
- Cache Go modules, NuGet packages and `panel/node_modules` so a run stays short.
- Remove the "There is no CI yet" comment from the root `justfile`, and make README prerequisites match what the workflow installs.

The Renki.Plugin build stays skipped in CI until TKT-01M6DCFST1's required-plugin child decides how CI compiles it. This ticket must not hide that skip; the required-plugin child makes it a failure.

### Files

- `.github/workflows/ci.yml` (new)
- `justfile` (the comment at the top, and `ci`)
- `AGENTS.md` ("`just ci` is the gate" paragraph) only if the wording changes
- `README.md` prerequisites

### Verification

- `just check ci` passes locally before the workflow is added.
- The workflow's first run on the pull request that adds it is green, and its log shows every component's `ci` and every `vm::test-*` recipe ran.
- A throwaway commit on a scratch branch that plants a Go compile error turns the run red. Link both runs in a note (this is goal 4.1's evidence).

## Acceptance criteria

- [ ] A workflow runs just ci on every pull request and on pushes to main
- [ ] The workflow installs Go, .NET 8, Node, Playwright Chromium, Python 3, just and git-ticket, and its first run is green
- [ ] A planted Go compile error on a scratch branch turns the run red, and both runs are linked in a note
- [ ] The root justfile no longer says there is no CI
