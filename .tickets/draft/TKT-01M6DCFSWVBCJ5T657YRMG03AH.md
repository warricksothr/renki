---
schema: 4
id: TKT-01M6DCFSWVBCJ5T657YRMG03AH
title: "Docs: README status, prerequisites, and recipes that bypass just"
type: chore
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - docs
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
created_at: 2026-11-02T05:42:34Z
updated_at: 2026-11-02T06:11:19Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-review
  name: ""
extensions: {}
---

## Description

Documentation and recipe drift.

### README

- The Status section says "Milestone 1 is written and waiting to be tried". In fact M1 (TKT-01M61T36B) is done, and M2, M3 and the panel epic are in progress.
- The diagram leaves out the panel, video and OBS.
- The prerequisites leave out npm and `just panel ui-install` (Playwright Chromium, needed by the panel `ci`), and Python 3 (broadcast, vm and streaming tests).

Rewrite Status to point at docs/ROADMAP.md, which the planning PR adds, rather than restating it.

### Recipes that go against AGENTS.md "Tasks"

- The root `package` recipe calls `dotnet build Renki.Plugin` directly. Route it through a `just plugin` recipe.
- `plugin/justfile:29-34`: `test` calls `go build` directly instead of `just controller …`. Its temp directory is not removed when the build or tests fail, because `set -e` exits before `rm -rf`. Use a `trap`.
- `scripts/streaming/windows.ps1`: in the `stop` and `codec` actions, the second `obs-adapter -action standby` call does not check `$LASTEXITCODE`.

## Acceptance criteria

- [ ] README status, diagram and prerequisites match the project
- [ ] package and plugin test go through recipes and clean up on failure
- [ ] windows.ps1 checks every obs-adapter exit code

## Notes

**agent:claude-code/renki-review** at 2026-11-02T06:11:19Z

The roadmap PR rewrote the README Status section to point at docs/ROADMAP.md. Still open: the diagram (panel, video, OBS), the prerequisites (npm, Playwright via just panel ui-install, Python 3) and the recipe items.
