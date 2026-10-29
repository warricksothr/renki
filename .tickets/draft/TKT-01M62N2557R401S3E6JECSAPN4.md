---
schema: 4
id: TKT-01M62N2557R401S3E6JECSAPN4
title: "M3: the bot survives danger and stands with its owner"
type: epic
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
assignees: []
milestone: M3
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T01:40:45Z
updated_at: 2026-10-29T13:23:51Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

M3 makes the bot safe to take out and useful in a fight. On 2026-10-28 it was killed twice by creatures while standing about and drowned once on a trip, and it never fought back or ran. M2 makes it work; M3 keeps it alive while it does, and lets it help.

The plugin's safety switch for this (Safety.AllowCombat) is off by default and stays that way until an owner turns it on.

## Acceptance criteria

- [ ] Hurt and outmatched, the bot retreats toward its owner or home instead of standing still
- [ ] With combat allowed, it fights what attacks it or its owner, and stops when told
- [ ] It remembers where it was hurt and avoids those places when it can

## Implementation plan

Order of work, set 2026-10-29: flee and retreat when hurt and outmatched (TKT-01M62N256); danger zones from where it was hurt (TKT-01M62N25C); attack with block and dodge behind Safety.AllowCombat, off by default (TKT-01M62N258); guard the owner and defend itself (TKT-01M62N25A). Revenge (TKT-01M62VXZH, TKT-01M62VXZJ) is outside M3's criteria and waits.
