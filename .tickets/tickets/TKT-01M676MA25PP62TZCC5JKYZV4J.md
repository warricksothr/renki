---
schema: 4
id: TKT-01M676MA25PP62TZCC5JKYZV4J
title: Land and verify the Preact panel stack
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
  actor: agent:codex/renki-milestones
  branch: t3/ui-stack-verification
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: b62448d99b07ef833ba2f9e5d3b05951b6fa6d72
  session: null
  claimed_at: 2026-10-30T20:04:54Z
  expires_at: null
archive: null
created_at: 2026-10-30T20:04:44Z
updated_at: 2026-10-30T20:06:52Z
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

- [ ] Combined UI stack passes full CI and committed review
- [ ] Both UI PRs are merged and deployed with verified package checksums
- [ ] Deployed panel camera, map and offline controls are verified

## Implementation plan

Review PR104 browser runner/scenarios and actual artifacts, integrate main into the top UI branch, retain both browser CI and camera-check CI. Run just check ci on the combined tree, commit integration and run just check review, then update the top PR. Merge PR102 first, retarget PR104 to main, and merge the reviewed integrated top. Deploy via existing VM recipe, inspect actual panel assets in native browser, verify live JPEG/map and offline fog/disabled actions, and keep Tas safe/offline afterwards. Record receipts and close this scoped ticket; continue existing M2/M3 tickets without counting fixture tests as game outcomes.

## Notes

**agent:codex/renki-milestones** at 2026-10-30T20:06:52Z

Integrated current main638e90e into the top UI branch. The only content conflict was docs/TESTING.md; retained both camera-check and deterministic-browser instructions. Root routing keeps panel/controller/plugin and vm::test-camera-check. Full combined just check ci passed (.checks/20261030T200454Z-awoG0E.log), including30 browser cases and both sizes of shared fixture assertions, plus controller/plugin checks.

Parent inspected mobile overview/settings and desktop chest-content hover screenshots, with no blocking visual issue. Partial review is .checks/panel-ui/parent-visual-review.json; uninspected screenshots remain unreviewed. Small-phone settings use horizontal scrolling with keyboard focus coverage; reducing that scroll is optional polish. Fixture success is not production auth/live camera/game evidence. Next: committed review, original stack merges and deployment verification.
