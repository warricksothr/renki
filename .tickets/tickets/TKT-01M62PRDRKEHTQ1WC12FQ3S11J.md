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
updated_at: 2026-10-29T22:29:16Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

`interact` (7.2): walk to an entity by id and use it as a player would (its Interactable, with `alt` and `hold`), behind the safety switch that fits what it is. The building block for claiming a bed, emptying a tombstone on a corpse run, and later levers and the like. Reports what the game answered, including refusals (a ward, a bed someone else owns, "you need a roof").

## Acceptance criteria

- [x] interact walks to an entity and uses it, with alt and hold
- [ ] A refusal comes back as a failure with the game's reason, not a success

## Implementation plan

Plugin: InteractTask walks to the entity (within 2 m, on its level, or 2.5 m of its collider), faces it, and calls its Interactable.Interact(player, hold, alt); hold presses once, then holds for 1 s as a held key repeats. A Harmony postfix on Player.Message records what the game says during the call, localized. True is success (detail carries what the game said); false fails with a new reason 'refused' whose detail is the game's message. Core InteractRule decides by kind: boss altars need AllowBossSummoning, ship controls AllowSailing, pickables AllowGather; drops, containers, crafting stations, portals and signs are refused as invalid_target, since they have their own tasks or open a window the bot cannot use. Wards: PrivateArea.CheckAccess refuses not_allowed. Safety: interact needs AllowMovement. Go: TaskInteract, InteractArgs, ReasonRefused, KnownTasks; docs 6.6, 7.2.

## Notes

**agent:claude-code/renki-groom** at 2026-10-29T15:08:48Z

Landed on t3/m2 and deployed on 2026-10-29.

- **Plugin:** InteractTask, with an allow-list in InteractRule. Allowed: beds and doors; your own grave when all of it fits; and, behind their switches, offering bowls and pickables. Wards, chests, stations, portals, signs, helms and anything unnamed are refused. A refusal comes back as the new reason `refused`, carrying the game's own words.
- **Chat:** new commands "go to bed" (only the bot's own bed), "claim the bed", "get/open/close the door".

**Checked in the game:** "get the door" from the panel walked the bot to the Wood Door and opened it ("used the Wood Door"). That ticks criterion 1.

**Not checked:** criterion 2, a refusal coming back with the game's reason. It is in the code and tested in chat (`It won't: ...`), but not yet seen in the game. Both beds in the base belong to someone else, and claiming one is the owner's call. Try "claim the bed" on one of the players' beds: expect "the Bed did nothing" (the game says nothing for a bed someone else owns), or "You need a roof" on an exposed bed.

**On the VM:** I added `interact` to allowedTasks through the panel settings.

**agent:codex/renki-resume** at 2026-10-29T22:29:16Z

Manual sleep/claim slice implemented: sleep/go to sleep/go to bed chooses the bot's own visible bed, otherwise a confirmed unclaimed nearby bed. Nullable owned metadata prevents inferring free ownership from old/unknown data. interact sleep:true claims/sets spawn, waits at most three seconds for ownership, then uses the bed again. Only actual attachment to that bed establishes in-bed success. Game checks/refusal messages are retained, including night/fire/roof/wet/enemy conditions. State distinguishes in_bed from actual sleeping. Unknown/taken selection, aliases, owner gating, incompatible args, shared wire fixtures and asynchronous ownership are covered by the gate (175 C# tests plus controller checks).

Live at 22:22:51–22:22:52Z: sleep selected explicitly free bed 1:38548, set spawn, confirmed ownership, and attempted the second use. The game refused "You can't sleep at this time" (day 51, fraction about 0.34). Perception then showed mine:true,owned:true; the other bed 1:38536 remained another player's. Evidence .checks/audit-window-6vVaDr.jsonl. The claim succeeded, but the broad PlayerBase effect-area trigger wrongly counted as interaction reach, so Tas had not approached it.

Fixed interaction reach to ignore trigger colliders and trigger ray hits. After full deployment, go to sleep at 22:28:17–22:28:19Z actually walked Tas from (-36.53,36.09,7.96) to (-32.03,36.09,11.97), close enough to the physical bed collider, and relayed only "You can't sleep at this time" as a refused task. HP25 throughout. Evidence .checks/audit-window-GuhIsv.jsonl. Sent him back to configured home afterward. No player's bed was reset and no death/time-of-day was staged.

Only acceptance criterion 1 is complete. Criterion 2 includes actual respawn at the bed, which has not been tested. Automatic claiming on home/join/respawn, persistent bed status and automatic no-free-bed notices remain unimplemented; nighttime positive sleep and roof/fire/wet/enemy/ward staging remain live checks. Owner stop/death/reconnect uses the existing task cancellation; there is no scheduled continuation after cancellation. Do not call the entire bed ticket or M2 complete.
