---
schema: 4
id: TKT-01M62PRDSBCRCPH5S0TGGCMAEY
title: Claim a bed near home, or tell the owner there is none
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - plugin
  - chat
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62PRDRKEHTQ1WC12FQ3S11J
  - TKT-01M62BMD18K3SE1WXJ7FDN8EPT
  - TKT-01M62EP0PDAB02DNJAJ6WS3MYD
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T02:10:24Z
updated_at: 2026-10-29T02:10:24Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Nimah, 2026-10-28: the bot should claim a bed close to home if one is available, and if not let the owner know. A claimed bed is also where the game respawns it: today it wakes at the world spawn after every death, and walks home from there.

### What to do

- **Perception (6.2):** beds also report whether someone owns them (`owned`), beside the existing `mine`, so the controller can tell a free bed from one that is taken.
- **The controller looks for a free bed** within a short distance of home (from perception when there, from memory otherwise). It claims the nearest by walking to it and interacting, which sets the bot's spawn point.
  - **When:** on `set home`, when it has no bed of its own on joining a world, and after a respawn far from home.
  - **When the game refuses** (no roof, too wet, a ward), it moves on to the next bed or reports the reason.
- **No free bed:** it tells an owner once, plainly, for example "There's no free bed near home. If you build me one, I'll claim it." It does not nag; it says it again only when asked (status), or when home moves.
- **Status** says whether the bot has a bed and where.

## Acceptance criteria

- [ ] Perception says whether a bed is owned, and 6.2 and the fixtures say so
- [ ] With a free bed near home, the bot claims it and respawns there after a death
- [ ] With none, it tells an owner once and says so in status
- [ ] A refusal from the game (no roof, a ward) is reported with its reason
