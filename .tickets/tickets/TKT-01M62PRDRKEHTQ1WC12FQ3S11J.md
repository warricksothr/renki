---
schema: 4
id: TKT-01M62PRDRKEHTQ1WC12FQ3S11J
title: "Plugin: interact task for beds, tombstones and the like"
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - plugin
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/m2
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 50de5da79868b41327287f748ee2061a1797aa05
  session: null
  claimed_at: 2026-10-29T14:19:23Z
  expires_at: null
archive: null
created_at: 2026-10-29T02:10:24Z
updated_at: 2026-10-29T14:19:23Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

`interact` (7.2): walk to an entity by id and use it as a player would (its Interactable, with `alt` and `hold`), behind the safety switch that fits what it is. The building block for claiming a bed, emptying a tombstone on a corpse run, and later levers and the like. Reports what the game answered, including refusals (a ward, a bed someone else owns, "you need a roof").

## Acceptance criteria

- [ ] interact walks to an entity and uses it, with alt and hold
- [ ] A refusal comes back as a failure with the game's reason, not a success

## Implementation plan

Plugin: InteractTask walks to the entity (within 2 m, on its level, or 2.5 m of its collider), faces it, and calls its Interactable.Interact(player, hold, alt); hold presses once, then holds for 1 s as a held key repeats. A Harmony postfix on Player.Message records what the game says during the call, localized. True is success (detail carries what the game said); false fails with a new reason 'refused' whose detail is the game's message. Core InteractRule decides by kind: boss altars need AllowBossSummoning, ship controls AllowSailing, pickables AllowGather; drops, containers, crafting stations, portals and signs are refused as invalid_target, since they have their own tasks or open a window the bot cannot use. Wards: PrivateArea.CheckAccess refuses not_allowed. Safety: interact needs AllowMovement. Go: TaskInteract, InteractArgs, ReasonRefused, KnownTasks; docs 6.6, 7.2.
