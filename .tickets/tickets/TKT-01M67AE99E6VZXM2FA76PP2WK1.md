---
schema: 4
id: TKT-01M67AE99E6VZXM2FA76PP2WK1
title: Balance panel monitoring layout and surface routine controls
type: task
status: in-progress
status_reason: null
priority: normal
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
  actor: agent:codex/panel-layout
  branch: t3/improve-control-panel-layout
  worktree: /home/sothr/.t3/worktrees/renki/t3-6f0693b4
  commit: 7700a93c992f8d73790e1489181354ffb083733a
  session: null
  claimed_at: 2026-10-30T21:11:47Z
  expires_at: null
archive: null
created_at: 2026-10-30T21:11:21Z
updated_at: 2026-10-30T21:21:31Z
created_by:
  id: agent:codex/panel-layout
  name: ""
updated_by:
  id: agent:codex/panel-layout
  name: ""
extensions: {}
---

## Description

Improve the desktop monitoring hierarchy from the supplied screenshot using local recorded fixtures only. Preserve map, camera, goals, safety controls and responsive accessibility. No VM, deployment or live game changes.

## Acceptance criteria

- [x] Balanced map and camera presentation removes the map-side dead space and surfaces routine commands and goals.
- [x] Deep preferences and editors remain accessible without crowding monitoring; desktop and mobile have no clipping or overlap.
- [x] Existing browser automation includes layout regressions and repeatable visually inspected screenshots.
- [ ] just check ci and post-commit just check review pass; open a linked PR without merging or deploying.

## Implementation plan

Make Map and Camera siblings in one equal-height desktop monitoring row; retain shared renderer and camera state handling. Extract the existing say form into a compact command bar by current status. Use a two-column activity/goals row directly below the visuals and a separate wider two-column references row, with inventory and place editor initially collapsed. Keep safety dock, keyboard behavior, disclosure navigation and mobile document scrolling. Extend recorded-fixture automation with geometric regression checks at desktop/mobile and a wide desktop stress case; inspect screenshots for normal, expanded, stale and offline states. Build embedded assets, run just check ci, commit, run just check review, resolve findings, then open/link a main-targeting PR. No live checks or deployment are needed for these layout-only changes.

## Notes

**agent:codex/panel-layout** at 2026-10-30T21:16:40Z

Implemented a balanced map/camera row with flexible canvas height, a command bar at current status, direct goal access, two-column Activity/Kit and separate reference cards. Inventory and place editing start collapsed. At 1100px and below the views stack, avoiding narrow letterboxed cameras; at 760px and below all cards stack and the safety dock remains fixed. Existing handlers, scope checks and renderer are unchanged.

Focused just check panel ui-test passed after updating disclosure interactions. The layout-monitoring scenario covers 1440x900, 390x844, 1920x1080, 1024x1080 and 768x1080, expanded view details, keyboard inventory focus and place editing. Visual inspection caught a preexisting fixture-only mobile header offset and an anchor screenshot timing problem; the banner now offsets only the sticky desktop header and anchor checks wait for header clearance. Intermediate captures are in .checks/layout-final; final gate artifacts will be .checks/panel-ui. No VM calls or live game checks were made or are required for this layout-only change. Aesthetic acceptance remains for the parent/user; do not merge or deploy this branch.

**agent:codex/panel-layout** at 2026-10-30T21:21:31Z

Full just check ci passed in .checks/20261030T211714Z-7eUrl3.log: 32 browser scenarios / 87 named checks with zero browser errors, all controller checks and 367 plugin core tests. The existing plugin DLL gate skips without Valheim assemblies; no live acceptance is claimed. A second gate run includes the final screenshot positioning improvement so expanded card titles remain clear of the fixture banner.

CI initially exposed an occupied default fixture port. panel ci now invokes its existing ui-test recipe with --port 0, allowing isolated worktrees to run concurrently without touching the other listener. Final screenshots/diagnostics are in .checks/panel-ui; visual-review-agent.json records the inspected subset and leaves user aesthetic acceptance pending. Reproduce layout artifacts with just check panel ui-test --scenario layout-monitoring --port 0 --output ../.checks/layout-review. Review and PR creation remain acceptance criterion 4.
