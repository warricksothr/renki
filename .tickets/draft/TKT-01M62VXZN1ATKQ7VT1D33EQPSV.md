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
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M62N12AYHHGR5W6D9RDEQKVM
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:49Z
updated_at: 2026-10-29T03:40:49Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

PickupTask counts success as the pack gaining the drop's item. If the bot
gains the same item from somewhere else at the same time (auto-pickup of
another stack nearby, which is usually fine), that counts too, and so does
the case where the bot uses an item before the pickup lands. Accepted on the
pickup branch as harmless. Fix if a report says it took something it did
not.

## Acceptance criteria

- [ ] pickup reports only what came from its own drop
