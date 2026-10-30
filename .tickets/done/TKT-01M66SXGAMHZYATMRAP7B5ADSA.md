---
schema: 4
id: TKT-01M66SXGAMHZYATMRAP7B5ADSA
title: Report menu after a requested logout finishes
type: bug
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - vm
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
created_at: 2026-10-30T16:22:34Z
updated_at: 2026-10-30T17:16:18Z
created_by:
  id: agent:codex/renki-atlas
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

During atlas restart checks just vm stop took about 68 seconds despite eventual successful game/controller shutdown. Code inspection found RenkiPlugin.Phase returns logging_out before checking whether Player.m_localPlayer and ZNet.instance have disappeared; loggingOut stays true after a requested logout. The VM stop script waits for menu, so an already completed native logout may exhaust its 60-second wait. Confirm this with a focused live trace, then make menu observable after logout without re-enabling automatic rejoin. This issue did not prevent atlas retention/offline checks.

## Acceptance criteria

- [x] Phase reports logging_out only during shutdown and menu once the game has returned there
- [x] AutoJoin suppression remains active after an explicit logout
- [x] A live logout/stop check recognizes menu without exhausting the 60-second wait

## Implementation plan

Publish phase transitions after explicit logout even though body control has ended. Resolve menu once Player and ZNet are gone before returning logging_out; keep the logout latch and AutoJoin suppression, including across a new welcome, so final reporting cannot restore automation. Verify the transition policy and controller handling, then confirm native logout reaches menu without the 60-second wait. This addresses the final-state issue also described by TKT-01M64Z4P80P9W2QSGQ22QAE56W — Logout: publish an accurate final connection and phase state; close that duplicate only after the shared behavior is verified.

## Notes

**agent:codex/renki-milestones** at 2026-10-30T17:16:18Z

Landed the phase/control correction in PR #83 (61c2ea5) and the already-logging-out VM wait in PR #84 (8ea7742); both deployed after CI and committed review. The real game audit .checks/audit-window-xggaKd.jsonl observes explicit logout at 16:56:20.421Z, logging_out, and menu at 16:56:21.859Z, about 1.44 seconds later. Cleanup stopped the game without the former 60-second phase wait and left the panel online. Phase transitions continue after body control ends; the logout latch blocks Welcome from restoring control, and AutoJoinPatch.Suppress remains set. ObservationScheduleTests cover the menu transition without resetting that latch. Native encounter evidence is in .checks/combat-trial-20261030T165304312270Z/summary.json. This also addresses the stale final state described by TKT-01M64Z4P80P9W2QSGQ22QAE56W — Logout: publish an accurate final connection and phase state.

## Summary

Landed/deployed PRs #83 and #84. Real logout reaches menu in about 1.44 seconds; reporting continues after control revocation, reconnect cannot restore automation, and VM stop waits for an already requested logout. CI, committed review and phase-policy checks pass.
