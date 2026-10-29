---
schema: 4
id: TKT-01M62N12HJW32JJTD2M3PEAEA4
title: "Controller: loot and deliver-when-full behaviours for hauling"
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62N129K1RGB8VSJZ3SDAC7N
  - TKT-01M62N12AYHHGR5W6D9RDEQKVM
  - TKT-01M62N12C9EWXCD54WMBH08T4E
  - TKT-01M62EP0PDAB02DNJAJ6WS3MYD
  - TKT-01M62N12G98VW2CYW94FG5RCTB
  - TKT-01M62G71QCVNG3DVW1HZ8HXA23
  - TKT-01M62VXYSN4RQV75HTA3BSD162
  - TKT-01M62VXYY6F094WXQS3FBQ57RF
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-resume
  branch: t3/collect-stow
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: a28ae9f0875b08ffcb291a17ee03f366ed4df8c6
  session: null
  claimed_at: 2026-10-29T21:47:34Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-29T21:54:36Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

Scene 1 of the M2 benchmark, built as two instances on the behaviour engine
rather than a hand-written routine (reshaped 2026-10-28, see the behaviours
epic TKT-01M62VXYQB6).

- **loot** (template `loot`, on by default): picks up drops near the bot,
  or near the owner it follows, while there is room. It leaves alone what an
  owner dropped on purpose: a drop that appears at an owner's feet with no
  felling, splitting or breaking nearby counts as deliberate for a few
  minutes.
- **deliver when full** (a `stow` instance, `items: *`, triggered by
  inventory full): carries the load to the chests at home, then goes back to
  the owner or the site. Choosing a ship or a waypoint instead comes later
  (TKT-01M62VXZ1JG9).

"help me" / "collect wood" (and the voice) switch both on for the owner who
asked. "stop" ends the current run. It says briefly what it is doing ("Full.
Taking this home.") and what stops it: no chest, chests full, nothing to pick
up. Owners only.

## Acceptance criteria

- [ ] loot runs as an instance, on by default, and picks up drops near the bot or its leader while there is room
- [ ] loot leaves an owner's deliberate drops alone
- [ ] Full, deliver-when-full stows the load in the chests at home and goes back
- [ ] It says when it cannot go on (no chest, chests full), and stop ends the run at once

## Implementation plan

The owner approved continuing toward collection/delivery and raised tool crafting/progression as follow-up. Start with two bounded, manually configured templates on the existing behaviour engine: stow from the pack into explicitly ordered chests with keep counts and exact transfer receipts; collect nearby live pickables/drops of selected prefabs inside an explicit world/site radius, stop below carry capacity, stow, and optionally return to the site. No default automation or follow detours in this slice. Leave the broader hauling acceptance criteria open for work-site inference, deliberate-drop detection, chat controls and leader resumption.

Add additive resource output metadata from the game's pickable/drop item data so flint and loose stones can be selected by output prefab and checked against weight/slot limits without guessing from scenery names. Unknown yields are ineligible for automatic collection. Never swing at trees or mine rocks in this slice. Preserve equipped gear, other gear, and requested keep counts. Current scoped observations and task-matched chest receipts govern the next step; stale memory is only a location hint. Bounded recoverable chest refusals/full results may try the next named chest; owner stop, permission loss, death and reconnect still revoke the complete run.

Stow reports remaining counts/reasons through behaviour status and a visible storage-needed panel message; repeated unchanged reports are suppressed. Test partial/full/busy/missing chests, changed contents, keep counts, protected gear, stale observations, capacity limits, interrupted runs and return routing. Run focused just checks, just check ci, committed just check review, then land/deploy. Live-check only non-destructive nearby pickables and Tas's test seeds using exact before/after inventory counts; defer missing staged checks explicitly. Record crafting/tool acquisition, repair and upgrade planning as draft follow-up instead of making axe crafting a prerequisite for testing flint/stone collection.

## Notes

**agent:codex/renki-resume** at 2026-10-29T21:54:36Z

Implemented the bounded manual collection/delivery slice on t3/collect-stow. Stow selects pack prefabs/groups, protects all gear and keep counts, approaches up to 16 explicitly named chests, and uses task-matched transfer receipts plus updated pack snapshots for each subsequent deposit. Expected full/busy/missing/unreachable chest results can fall back; stop and permission failures remain terminal. A copied, world-scoped storage-needed status is visible on the panel and deduplicated in-game announcements; reports and run state are controller-session state.

Collect selects only current pickable/drop output prefabs inside an explicit world/site radius, reserves slots and carry weight (80% default, at most 90%), stows selected carried materials above keep counts, and optionally returns to the site. It has bounded targets, steps and five-second fresh-observation waits. Metadata comes from actual game pickable/drop items, includes possible bonuses and world resource scaling, and excludes unknown extra/theft-provoking products. PickableItem perception is added. Existing deliberate drops require explicit drops:true; there are no default jobs, work-site inference or leader detours/resumption. Trees, mining and crafting are not implemented by this slice.

Focused Go regressions passed, including partial/full/busy/missing chest policy, exact receipts/pack ordering, reserves, protected gear, yield/site selection, capacity, pickable-to-drop-to-stow-to-return, bounded observation waits, duplicate shortfalls and bridge-level owner-stop preemption. Full just check ci passed: race suite, Linux/Windows builds, JS map checks, plugin build and 166 C# tests including the new shared yield fixture. The storage-needed panel renders on desktop/mobile without console errors or overflow. Committed review, landing, deployment and a controlled live collection/seed check follow. Keep acceptance criteria open until evidence is recorded; broader hauling remains incomplete.

Filed draft TKT-01M64T4WGZA175B38A8GBQ9YD4 (Tools: acquire, craft, repair and upgrade harvesting equipment) for the owner's follow-up: prefer carried/stored tools and repair, then plan recipes/stations/material shortages, obey resource/progression policy for upgrades, and resume the interrupted gathering job. No crafting work was promoted.
