---
schema: 4
id: TKT-01M673HPNRCJ7DC62RWPJKPZE4
title: Redesign the control panel around a Preact workspace
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - tooling
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/panel-redesign
  branch: t3/redesign-preact-control-panel
  worktree: /home/sothr/.t3/worktrees/renki/t3-f8a812b0
  commit: 5a5f5c9cf4ad68b69b1fa11c766846cb9646110c
  session: null
  claimed_at: 2026-10-30T19:11:01Z
  expires_at: null
archive: null
created_at: 2026-10-30T19:10:53Z
updated_at: 2026-10-30T19:27:56Z
created_by:
  id: agent:codex/panel-redesign
  name: ""
updated_by:
  id: agent:codex/panel-redesign
  name: ""
extensions: {}
---

## Description

Implement the explicitly requested Preact desktop/mobile redesign in an isolated draft PR using read-only fixtures; preserve all panel controls and the shared map renderer without VM work.

## Acceptance criteria

- [x] Preact workspace exposes health, food, goals, camera and safe actions with responsive accessible controls.
- [x] All existing controls, map layers, offline atlas and exact snapshot/preview contracts remain available.
- [ ] Reproducible frontend recipes integrate builds, package and CI; fixture checks and review evidence are recorded.

## Implementation plan

Build a Preact workspace around the existing map and control adapters, keeping one renderer and API contract. Promote camera to an always-visible scoped image with explicit freshness; summarize timeline records with expandable originals. Add locked frontend tooling and component routing, read-only preview fixtures, behavior and responsive checks; run CI, commit, review, and publish only a draft PR.

## Notes

**agent:codex/panel-redesign** at 2026-10-30T19:27:56Z

Implemented the scoped Preact control room; rebased onto main cf59e5d including PR #99 camera CSP fix. One existing map renderer powers panel, exact exports and standalone previews. Camera uses authenticated blobs with image load/decode verification, generation/scope guards and explicit disabled/pending/offline/failed/stale presentation. Document scrolling plus bounded timeline/settings/goals/place panels prevents clipped controls. All legacy controls and runtime settings remain; inventory now reads the scoped API snapshot after replay to avoid clearing the initial pack.

Evidence: just check ci passed (.checks/20261030T192542Z-ixfnhS.log, 28 seconds; plugin DLL skipped without game assemblies). Native recorded-fixture browser checks passed 33 assertions at 1440x900 and 390x844; mobile report .checks/panel-mobile-checks.json. Actual img naturalWidth960/naturalHeight540, no uncaught errors, 134 captured terrain tiles; grouped chest hovers retain known and unknown contents. Standalone map-only output .checks/panel-map-only.html, full replay .checks/panel-recorded.html. Design/parity/build rationale in docs/PANEL-REDESIGN.md. Raw embedded asset growth 16204 bytes, measured gzip growth 6286 bytes. No VM/game actions/deployment; parent owns live verification. Local commit review and draft PR publication remain.
