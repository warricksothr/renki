---
schema: 4
id: TKT-01M6H243JB6S4MBXVKZD89WQKG
title: Close doors Tas opened after passing through the doorway
type: bug
status: review
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - movement
  - live
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
created_at: 2026-11-03T15:58:23Z
updated_at: 2026-11-03T16:40:46Z
created_by:
  id: agent:codex/home-survey-delivery
  name: ""
updated_by:
  id: agent:codex/home-survey-delivery
  name: ""
extensions: {}
---

## Description

The owner observed double doors left open after Tas passed through. Automatic walker opening is tracked, but explicit interact tasks bypass Doorman registration. Existing cleanup also waits until 3.5–4.5m from the hinge and drops tracking before asynchronous closing is confirmed.

Track each initially closed leaf opened by Tas across both interaction paths, close nearby leaves after clearing their doorway plane, and retain bounded confirmation/retry without toggling during network settlement. Leave already-open doors alone and preserve ward/key/AllowDoors behavior.

## Acceptance criteria

- [x] Walker and explicit interaction openings register each initially closed door leaf; initially open doors are not adopted.
- [x] Door cleanup recognizes passage near both hinges and persists across task completion, preserving floor separation and configuration.
- [x] Closing waits for replicated state, retries only within a bounded settlement budget, and records refusal rather than claiming closure.
- [ ] Focused regressions, full CI and committed review pass; native double-door opening, passage and verified closure are retained.

## Implementation plan

Add a pure per-leaf passage/settlement lifecycle used by the native Doorman, register explicit Door interactions as well as walker openings, and close after clearing the doorway plane while still nearby. Keep closure pending until an observed closed state, allow one bounded retry after settlement, and trace persistent refusal. Regress paired hinge geometry, pre-opened/failed interactions, idle completion, floor separation and asynchronous state. Run focused plugin checks, the full gate, committed review and checksum-verified deployment; native evidence remains separate from offline policy.

## Notes

**agent:codex/home-survey-delivery** at 2026-11-03T16:03:42Z

Implemented shared registration for explicit and walker opening, per-leaf doorway-plane passage, idle cleanup, observed closure and at most two close requests separated by 3s. Already-open, denied, hold-use and disabled registrations are excluded; different floors/out-of-reach and unclosable leaves do not trigger closure. just check plugin test-core DoorPassageTests passed (.checks/20261103T155934Z-yfwTM9.log), just check fmt passed and full just check ci passed162s (.checks/20261103T160045Z-9rgBXg.log). Native double-door evidence is still pending; no native closure acceptance claimed.

**agent:codex/home-survey-delivery** at 2026-11-03T16:22:41Z

Added the double-door edge case where opening begins exactly on the doorway plane: cleanup recognizes clearing either side while still in reach. The existing per-leaf closure/replication budget remains unchanged. Combined offline DoorPassage regressions passed in .checks/20261103T161946Z-AGIa9e.log. Native double-door passage/verified closure remains unearned; Tas is emergency-offline during pursuit, so no exposed probe was attempted.

**agent:codex/home-survey-delivery** at 2026-11-03T16:40:45Z

Code landed in2e8c6d1. Full just check ci passed170s (.checks/20261103T163500Z-YLtrJQ.log); committed just check review passed8s with no findings (.checks/20261103T163822Z-KdkkbC.log). just check vm install passed16s with installed controller/Core/plugin checksum verification (.checks/20261103T163834Z-MJnAw1.log). just vm panel-start restored the independent panel without starting the game. Read-only delivered observation confirms offline; Loot remains disabled. The normal settings API added inspect_container while preserving all18 existing task permissions; readback confirmed it. Evidence .checks/new-home-delivered-observation.json, .checks/new-home-delivered-settings.json and .checks/new-home-delivered-behaviours.json. Native survival, double doors, chest inventories, survey/return and bed ownership remain unearned. No game rejoin or dangerous probe was attempted.

## Summary

Implemented, tested, reviewed and installed per-leaf explicit/walker door cleanup, double-hinge/starting-plane passage and bounded observed closure. AC4 remains unchecked pending native double-door opening/passage/closure proof. Tas is emergency-offline during an actual pursuit, so no exposed probe was run.
