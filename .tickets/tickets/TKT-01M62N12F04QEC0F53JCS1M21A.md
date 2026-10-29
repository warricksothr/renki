---
schema: 4
id: TKT-01M62N12F04QEC0F53JCS1M21A
title: "Plugin: harvest task: chop, split and break with the best tool"
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - movement
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62N129K1RGB8VSJZ3SDAC7N
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-resume
  branch: t3/harvesting
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 9f47b78ff635a07b66753ed259f564dc8b97b080
  session: null
  claimed_at: 2026-10-29T22:56:16Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-29T22:56:16Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

`harvest` (7.2): approach a tree, log, stump or rock by entity id, equip the best tool that meets its `minTool`, face it, and swing until it breaks; with `collect` (the default), sweep up the drops within a few metres afterwards.

A felled tree becomes a new `log` object and a `stump`; a log splits into halves, each a new object, and those break into wood (6.2). One harvest is one object; the controller chains them by watching perception for what appeared where the old one was. Swinging needs care with stamina, and with the falling trunk, which hurts.

Fails with `no_tool` when nothing in the inventory meets minTool, `inventory_full` when collecting cannot fit more, and `target_lost` when someone else breaks it first.

## Acceptance criteria

- [ ] harvest fells a tree, splits a log and breaks a stump or rock by id with the best tool it carries
- [ ] It collects the drops within a few metres unless collect is false
- [ ] It fails with no_tool, inventory_full or target_lost as 6.6 names them
- [ ] It does not stand under a falling trunk

## Implementation plan

Start with the explicitly approved supplied-axe benchmark, using the AxeFlint withdrawn from owner-designated rag chest 1:30592. Implement one bounded harvest body task through normal melee controls: select usable chop/pick damage and required tool tier, approach a visible resource collider on its floor, respect wards and nearby characters/buildings, face it, wait for stamina and settled logs, and stop attacks immediately when cancelled or interrupted. No fabricated HitData, inventory creation, or forced world damage.

Track real local hit/destruction evidence so an absent target does not silently become a successful harvest. Back away on felling and wait for falling logs to settle before any collection. Collect only new nearby item drops through existing pickup behavior with weight/slot checks and finite work. Keep harvest of a tree/log/stump/rock distinct from controller chaining of its descendants; establish the primitive first, then complete the supplied-tool chop/split/collect/deliver/return benchmark. Preserve seeds, rag gear and unrelated chest stock. Record unresolved live physics and multi-part rock coverage honestly.

Run focused policy/protocol checks, just check ci, committed just check review, then land/deploy and live-check a controlled target away from the home and players. Full/busy/warded/multi-destination storage checks remain separately tracked; the first extra home chest approach (1:30595) failed unreachable without moving a seed and needs a suitable approach point before retrying.
