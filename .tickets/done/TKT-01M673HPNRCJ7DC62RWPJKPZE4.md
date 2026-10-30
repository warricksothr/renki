---
schema: 4
id: TKT-01M673HPNRCJ7DC62RWPJKPZE4
title: Redesign the control panel around a Preact workspace
type: task
status: done
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
claim: null
archive: null
created_at: 2026-10-30T19:10:53Z
updated_at: 2026-10-30T19:38:41Z
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
- [x] Reproducible frontend recipes integrate builds, package and CI; fixture checks and review evidence are recorded.

## Implementation plan

Build a Preact workspace around the existing map and control adapters, keeping one renderer and API contract. Promote camera to an always-visible scoped image with explicit freshness; summarize timeline records with expandable originals. Add locked frontend tooling and component routing, read-only preview fixtures, behavior and responsive checks; run CI, commit, review, and publish only a draft PR.

## Notes

**agent:codex/panel-redesign** at 2026-10-30T19:27:56Z

Implemented the scoped Preact control room; rebased onto main cf59e5d including PR #99 camera CSP fix. One existing map renderer powers panel, exact exports and standalone previews. Camera uses authenticated blobs with image load/decode verification, generation/scope guards and explicit disabled/pending/offline/failed/stale presentation. Document scrolling plus bounded timeline/settings/goals/place panels prevents clipped controls. All legacy controls and runtime settings remain; inventory now reads the scoped API snapshot after replay to avoid clearing the initial pack.

Evidence: just check ci passed (.checks/20261030T192542Z-ixfnhS.log, 28 seconds; plugin DLL skipped without game assemblies). Native recorded-fixture browser checks passed 33 assertions at 1440x900 and 390x844; mobile report .checks/panel-mobile-checks.json. Actual img naturalWidth960/naturalHeight540, no uncaught errors, 134 captured terrain tiles; grouped chest hovers retain known and unknown contents. Standalone map-only output .checks/panel-map-only.html, full replay .checks/panel-recorded.html. Design/parity/build rationale in docs/PANEL-REDESIGN.md. Raw embedded asset growth 16204 bytes, measured gzip growth 6286 bytes. No VM/game actions/deployment; parent owns live verification. Local commit review and draft PR publication remain.

**agent:codex/panel-redesign** at 2026-10-30T19:38:41Z

Draft PR #102: https://github.com/warricksothr/renki/pull/102 (registered with T3, unmerged/undeployed). Final implementation rebased onto main 861c31c, preserving camera CSP and parent camera API changes. Supports effective config plus baseConfig/temporary override expiry; settings writes retain X-Renki-Act and existing permissions/pins. Native fixture checks now pass 35 assertions at 1440x900 and 390x844 (reports .checks/panel-desktop-checks.json and .checks/panel-mobile-checks.json); additional 768x1024 and 320px width checks have no page overflow. JPEG decoded at960x540,134 captured terrain tiles, known/unknown chest hovers preserved. Mobile safety dock stays fixed at viewport bottom while scrolling.

Supersedes earlier size/check-count notes: just panel measure reports total raw assets99764 ->117167 bytes (+17403), measured gzip29396 ->35986 bytes (+6590). Losslessly compressed fixture transport lets the recorded standalone map render in native html_preview (~166KB instead of513KB); production map semantics/data unchanged. Full workspace also rendered in html_preview under the image CSP. Camera scope/race/temporary tests and compressed fixture integrity tests pass in just check ci. Complete integrated gate .checks/20261030T193645Z-gDtBkC.log passed; just check review at51a9e10 passed with no findings (.checks/20261030T193717Z-3JPTNb.log).

Remaining review is the requested human design review of the draft. No VM/game actions, deployments, merges or edits to procedure-research PR #92. Parent retains live VM/native work. Local recorded fixtures are replay evidence, not live task-success evidence.

## Summary

Implemented and published draft PR #102, Redesign the Renki panel as a Preact control room: https://github.com/warricksothr/renki/pull/102. The responsive workspace preserves all controls and the one shared map renderer, exposes actual scoped camera imagery and temporary/base capture state, bounds diagnostics/editors, and integrates locked frontend builds into the shared just/Go embedding/CI workflow. CI and commit review pass;35 native fixture assertions pass at desktop/mobile sizes. Design/parity/size rationale is in docs/PANEL-REDESIGN.md. Requested implementation is complete; PR remains draft for human visual review, without merge/deployment or VM actions.
