---
schema: 4
id: TKT-01M62VXZJHS43K9ZHSWD3Z4VHT
title: Decide whether the bot may take revenge on a player
type: spike
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - controller
  - safety
assignees: []
milestone: M3
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies:
  - TKT-01M62VXZHFG5VECGXYPPMDHDSH
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

Raised on 2026-10-28: should the bot mark for death a player who kills its
owner? Not decided. If it is wanted, it is its own switch, off by default,
and the bot asks an owner before going after anyone, every time. Settle
whether it is wanted at all before building anything.

## Acceptance criteria

- [ ] An owner has decided whether this is wanted, and the decision is recorded here
