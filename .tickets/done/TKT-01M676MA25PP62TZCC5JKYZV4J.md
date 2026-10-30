---
schema: 4
id: TKT-01M676MA25PP62TZCC5JKYZV4J
title: Land and verify the Preact panel stack
type: task
status: done
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
claim: null
archive: null
created_at: 2026-10-30T20:04:44Z
updated_at: 2026-10-30T20:12:29Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Integrate owner-approved Preact redesign PR #102 and stacked UI automation PR #104 with current main, retain all newer runtime-camera tooling, run the complete combined gate/review, merge both PRs, deploy and verify live/offline panel behavior. Then resume remaining M2 harvesting and M3 combat checks; UI fixtures do not count as game milestone evidence.

## Acceptance criteria

- [x] Combined UI stack passes full CI and committed review
- [x] Both UI PRs are merged and deployed with verified package checksums
- [x] Deployed panel camera, map and offline controls are verified

## Implementation plan

Review PR104 browser runner/scenarios and actual artifacts, integrate main into the top UI branch, retain both browser CI and camera-check CI. Run just check ci on the combined tree, commit integration and run just check review, then update the top PR. Merge PR102 first, retarget PR104 to main, and merge the reviewed integrated top. Deploy via existing VM recipe, inspect actual panel assets in native browser, verify live JPEG/map and offline fog/disabled actions, and keep Tas safe/offline afterwards. Record receipts and close this scoped ticket; continue existing M2/M3 tickets without counting fixture tests as game outcomes.

## Notes

**agent:codex/renki-milestones** at 2026-10-30T20:06:52Z

Integrated current main638e90e into the top UI branch. The only content conflict was docs/TESTING.md; retained both camera-check and deterministic-browser instructions. Root routing keeps panel/controller/plugin and vm::test-camera-check. Full combined just check ci passed (.checks/20261030T200454Z-awoG0E.log), including30 browser cases and both sizes of shared fixture assertions, plus controller/plugin checks.

Parent inspected mobile overview/settings and desktop chest-content hover screenshots, with no blocking visual issue. Partial review is .checks/panel-ui/parent-visual-review.json; uninspected screenshots remain unreviewed. Small-phone settings use horizontal scrolling with keyboard focus coverage; reducing that scroll is optional polish. Fixture success is not production auth/live camera/game evidence. Next: committed review, original stack merges and deployment verification.

**agent:codex/renki-milestones** at 2026-10-30T20:12:29Z

UI stack merged: PR102 at0f5cc2d and PR104 at a525b18. Combined just check ci passed (.checks/20261030T200454Z-awoG0E.log), committed reviews passed (.checks/20261030T200653Z-eYwtHg.log and .checks/20261030T200824Z-Rn1vyf.log). After merging PR102, synchronizing the top branch with main changed no files; reviewed before push. GitHub briefly reported unknown mergeability while recalculating; the subsequent merge succeeded. just check vm install checksum-verified final a525b18 (.checks/20261030T200904Z-gMN0MM.log).

Native browser verified actual a525b18, live connected status, healthy69.1/69.1HP, decoded visible960x540 JPEG, scoped map export with80 tiles/worldUID827096491/connection1,32 inventory slots, enabled controls and zero horizontal overflow. Screenshot /home/sothr/.t3/userdata/browser-artifacts/browser-screenshot-100-101-221-77-mvveqi5b-0bd73b0b.png. Native JPEG .checks/camera-20261030T201148290562Z.jpg and shared-renderer map .checks/map-snapshot-20261030T201148335185Z/map.html captured. After logout/game stop and controller-only restart, browser confirmed camera offline/no image, retained80 map tiles and all game safety actions disabled. The initially verified offline map also retained98 saved tiles. These are positive deployed UI checks; no harvest or combat milestone claimed. Current normal camera interval reports5s. Tas is safe/offline, panel online. Return now to M2 controller gather/delivery and M3 encounter work.

## Summary

Merged PR102 and PR104 in order; installed a525b18 with checksum verification. Full combined CI,30 browser cases/shared regressions and committed reviews pass. Native browser proved visible decoded960x540 game camera, current scoped map export, inventory and controls while online, then cleared live images/disabled controls while retaining fogged terrain offline. Exact evidence and review limits are in the final note. UI stack is complete; M2/M3 remain separate ongoing work.
