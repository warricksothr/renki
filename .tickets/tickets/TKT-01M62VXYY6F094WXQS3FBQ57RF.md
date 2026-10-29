---
schema: 4
id: TKT-01M62VXYY6F094WXQS3FBQ57RF
title: "Controller: stow template: move all of some items to a list of chests"
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
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies:
  - TKT-01M62VXYSN4RQV75HTA3BSD162
  - TKT-01M62VXYWJHB5BC9VGCBM0PVXD
  - TKT-01M62N12C9EWXCD54WMBH08T4E
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-resume
  branch: t3/collect-stow
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: a28ae9f0875b08ffcb291a17ee03f366ed4df8c6
  session: null
  claimed_at: 2026-10-29T21:47:33Z
  expires_at: null
archive: null
created_at: 2026-10-29T03:40:48Z
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

"Move all of `items` from `from` to the chests in `to`, in order." This is
the core block of organising the base.

### Parameters

- `items`: prefabs, a group (`wood`, `ore`, `food`) or `*`
- `from`: the bot's pack for now. Chests and the ground in an area come later.
- `to`: an ordered list of chests. A resource with a lot of stock fills
  several, one after another.
- `keep`: how many of each item the bot holds back (food, a stack of wood
  for fires)
- `overflow`: `tell` (the default) or `leave`

### Running

1. Plan from chest memory: for each chest in order, top up partial stacks
   of the same item first, then use empty slots.
2. Walk to each chest, deposit, and read back what is inside. If a chest
   has less room than remembered, plan again from what is there.
3. Whatever is left is a shortfall.

### Reporting a shortfall

Tell the owners what did not fit and why: *"I need storage for 140 Stone:
stone-1 and stone-2 are full."*, or *"No chest is set for Resin; I have
30."* The panel shows a "storage needed" note until it is solved. No
nagging: the message repeats only when the shortfall changes or after a
long gap. The bot never throws anything away (Safety rule 8). When purging
is the answer, it says so and leaves the choice to the players.

## Acceptance criteria

- [ ] stow fills the listed chests in order, topping up stacks before using empty slots
- [ ] Several chests for one item fill one after another
- [ ] A chest with less room than remembered is planned again from what it really holds
- [ ] What does not fit is reported to the owners and on the panel, and the report repeats only when it changes
- [ ] Nothing is destroyed or dropped to make room

## Implementation plan

The owner approved continuing toward collection/delivery and raised tool crafting/progression as follow-up. Start with two bounded, manually configured templates on the existing behaviour engine: stow from the pack into explicitly ordered chests with keep counts and exact transfer receipts; collect nearby live pickables/drops of selected prefabs inside an explicit world/site radius, stop below carry capacity, stow, and optionally return to the site. No default automation or follow detours in this slice. Leave the broader hauling acceptance criteria open for work-site inference, deliberate-drop detection, chat controls and leader resumption.

Add additive resource output metadata from the game's pickable/drop item data so flint and loose stones can be selected by output prefab and checked against weight/slot limits without guessing from scenery names. Unknown yields are ineligible for automatic collection. Never swing at trees or mine rocks in this slice. Preserve equipped gear, other gear, and requested keep counts. Current scoped observations and task-matched chest receipts govern the next step; stale memory is only a location hint. Bounded recoverable chest refusals/full results may try the next named chest; owner stop, permission loss, death and reconnect still revoke the complete run.

Stow reports remaining counts/reasons through behaviour status and a visible storage-needed panel message; repeated unchanged reports are suppressed. Test partial/full/busy/missing chests, changed contents, keep counts, protected gear, stale observations, capacity limits, interrupted runs and return routing. Run focused just checks, just check ci, committed just check review, then land/deploy. Live-check only non-destructive nearby pickables and Tas's test seeds using exact before/after inventory counts; defer missing staged checks explicitly. Record crafting/tool acquisition, repair and upgrade planning as draft follow-up instead of making axe crafting a prerequisite for testing flint/stone collection.

## Notes

**agent:claude-code/renki-groom** at 2026-10-29T04:12:05Z

2026-10-28: 'full' is slots or weight. The bot is overloaded past weightMax (state 6.1: encumbered, and stranded once stamina runs out, when it cannot move at all). Deliver-when-full must trigger on weight near weightMax as well as on free slots, and loot must stop short of it.

**agent:codex/renki-resume** at 2026-10-29T21:54:36Z

Implemented the bounded manual collection/delivery slice on t3/collect-stow. Stow selects pack prefabs/groups, protects all gear and keep counts, approaches up to 16 explicitly named chests, and uses task-matched transfer receipts plus updated pack snapshots for each subsequent deposit. Expected full/busy/missing/unreachable chest results can fall back; stop and permission failures remain terminal. A copied, world-scoped storage-needed status is visible on the panel and deduplicated in-game announcements; reports and run state are controller-session state.

Collect selects only current pickable/drop output prefabs inside an explicit world/site radius, reserves slots and carry weight (80% default, at most 90%), stows selected carried materials above keep counts, and optionally returns to the site. It has bounded targets, steps and five-second fresh-observation waits. Metadata comes from actual game pickable/drop items, includes possible bonuses and world resource scaling, and excludes unknown extra/theft-provoking products. PickableItem perception is added. Existing deliberate drops require explicit drops:true; there are no default jobs, work-site inference or leader detours/resumption. Trees, mining and crafting are not implemented by this slice.

Focused Go regressions passed, including partial/full/busy/missing chest policy, exact receipts/pack ordering, reserves, protected gear, yield/site selection, capacity, pickable-to-drop-to-stow-to-return, bounded observation waits, duplicate shortfalls and bridge-level owner-stop preemption. Full just check ci passed: race suite, Linux/Windows builds, JS map checks, plugin build and 166 C# tests including the new shared yield fixture. The storage-needed panel renders on desktop/mobile without console errors or overflow. Committed review, landing, deployment and a controlled live collection/seed check follow. Keep acceptance criteria open until evidence is recorded; broader hauling remains incomplete.

Filed draft TKT-01M64T4WGZA175B38A8GBQ9YD4 (Tools: acquire, craft, repair and upgrade harvesting equipment) for the owner's follow-up: prefer carried/stored tools and repair, then plan recipes/stations/material shortages, obey resource/progression policy for upgrades, and resume the interrupted gathering job. No crafting work was promoted.
