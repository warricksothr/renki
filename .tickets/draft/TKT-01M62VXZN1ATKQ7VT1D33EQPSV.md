---
schema: 4
id: TKT-01M62VXZN1ATKQ7VT1D33EQPSV
title: "Pickup: counts the same item taken from elsewhere as success"
type: bug
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - plugin
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62N12AYHHGR5W6D9RDEQKVM
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:49Z
updated_at: 2026-11-02T15:57:27Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m2
  name: ""
extensions: {}
---

## Description

PickupTask counts success as the pack gaining the drop's item. If the bot gains the same item from somewhere else at the same time (auto-pickup of another stack nearby, which is usually fine), that counts too, and so does the case where the bot uses an item before the pickup lands. Accepted on the pickup branch as harmless. Fix if a report says it took something it did not.

### Why it may matter now

Goal 2.2 measures a pickup ratio from pickup receipts (the haul report, TKT-01M6DDFTDD). A receipt that counts another stack's items would inflate that ratio. If the haul report shows receipts larger than the drop's stack, fix this before trusting the ratio.

### Files

- `plugin/Renki.Plugin/PickupObservation.cs`, `plugin/Renki.Core/Movement/PickupTracking.cs`, `plugin/Renki.Plugin/BodyTasks.cs`
- `plugin/Renki.Tests/PickupTrackingTests.cs`

### Verification

- A test where another stack of the same prefab lands in the pack during the pickup, and the receipt reports only the target drop's count.
- `just check plugin test-core PickupTrackingTests`, then `just check ci`.

## Acceptance criteria

- [ ] pickup reports only what came from its own drop
