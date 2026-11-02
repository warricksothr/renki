---
schema: 4
id: TKT-01M6EFGK2M7MBNZRRJABMBCMGM
title: "Lint: analyzers for Renki.Core, and ruff for scripts"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - tooling
  - plugin
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
updated_at: 2026-11-02T15:54:40Z
created_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

Goal 4.1 asks for real lint for the plugin and the scripts. `just plugin lint` runs only `dotnet format whitespace`, `TreatWarningsAsErrors` is false in `plugin/Directory.Build.props`, and Renki.Core has no analyzers. About 6.6k lines of Python under `scripts/` (the VM trial supervisors and their tests) are not linted at all. Split out of TKT-01M6DCFST1 (CI: run just ci automatically, require the plugin, real lint), item 3, C# and scripts half.

### What to build

- Turn on .NET analyzers (`EnableNETAnalyzers`, `AnalysisLevel`) for `plugin/Renki.Core`, and make new warnings fail the build there. Leave Renki.Plugin's settings alone unless it compiles in CI (see the required-plugin child).
- Add a pinned `ruff check` (or pyflakes) pass over `scripts/` as a root recipe, for example `just scripts-lint`, and add it to the root `ci` list. It spans the repository, so it belongs in the root `justfile` (AGENTS.md, "Tasks").
- PowerShell under `scripts/vm/` and `scripts/streaming/`: PSScriptAnalyzer needs `pwsh`, which CI may not have. Either add it, or note in this ticket why it stays out.
- Fix what they report, or suppress each finding with a stated reason.

### Files

- `plugin/Directory.Build.props`, `plugin/Renki.Core/Renki.Core.csproj`, `plugin/justfile` (`lint`)
- `justfile` (a new scripts lint recipe, and `ci`)
- `scripts/*.py`, `scripts/vm/*.py` as flagged

### Verification

- `just check plugin lint`, `just check plugin build` and the new scripts lint recipe pass.
- `just check ci` passes.

## Acceptance criteria

- [ ] Renki.Core builds with .NET analyzers on and new warnings fail its build
- [ ] A root recipe runs a pinned ruff or pyflakes over scripts/ and just ci runs it
- [ ] PowerShell lint is added, or this ticket says why it stays out
