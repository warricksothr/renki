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
updated_at: 2026-10-29T22:26:17Z
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

**agent:codex/renki-resume** at 2026-10-29T21:59:57Z

The first live stow check on cf04d60 exposed a startup publication-order bug before any body task or inventory mutation: inventory seq 113 arrived at 21:57:21.842Z while state still said connecting/world empty; the first in_world/DizzysHellheim state seq 117 arrived at 21:57:22.005Z and correctly cleared observations from the old world. With an unchanged pack there was no second inventory. The stow job therefore failed its bounded five-second observation wait at 21:58:06Z. Tas remained idle at home, 25/25 HP; the 15 seeds were untouched.

Follow-up t3/world-observation-order fixes the sender, preserving the controller's world-clearing rule. ObservationSchedule forces state on phase/world/player-identity transitions even between periodic ticks. RenkiPlugin publishes that state before reporter events, inventory, terrain and perception, and resends observations on scope changes. Welcome resets the state deadline. Inventory is emitted only in-world/dead. Regressions cover entering between periodic ticks, reconnect, different worlds, replacement players and menu transitions. Focused just check plugin test-core ObservationScheduleTests passed; gate/review and deployment follow. The seed and loose-stone checks must be retried after the fix, and are not yet passes.

**agent:codex/renki-resume** at 2026-10-29T22:08:25Z

Live checks on 91b4835 after the startup fix:
- First in-world state seq117 at 22:02:57.890Z preceded inventory seq118 at 22:02:57.903Z. The unchanged 15-seed inventory was available without a reconnect or pack mutation.
- Stow t-1 at 22:03:12–22:03:14Z deposited exactly five BeechSeeds into 1:30592 and preserved ten. The chest's one ArmorRagsChest was unchanged. Withdraw t-2 at 22:03:33Z restored the pack to 15 seeds and the chest to its original contents.
- Initial collection move t-3 failed safely inside the house: local search exhausted 215 nodes /1720 probes at 22:03:45Z (47.72 ms total, 2.11 ms worst frame), with no item mutation. Explicit door interaction t-4 moved Tas from (-36.07,36.09,7.82) to (-37.69,36.09,9.52) and opened the west door. That combines opening and repositioning; it does not isolate the navigation cause.
- Retried stone collection then completed t-5 move to site, t-6 use Stone, t-7 take one Stone, t-8 approach chest, t-9 deposit exactly one Stone, t-10 return to site (22:04:58–22:05:12Z). Chest held Stone1 plus the unchanged rag armour. Tas stayed 25/25 HP. This is a controlled full loop after doorway staging, not proof of reliable unattended house exits.
- The next flint job used its pickable at 22:06:02Z and auto-pickup put Flint1 in the pack at 22:06:03.489Z. A stale perceived product caused pickup t-13 to arrive 97 ms later and be rejected invalid_target: no drop in range. The Flint is safe in the pack, but delivery stopped. No full flint-loop pass is claimed yet.

Follow-up t3/collection-pickup-race checks measured whole-yield gains before choosing stale product drops. A rejected missing product can be handled only while a known pickable's output is pending; fresh inventory evidence is still required. Added regressions cover both inventory-before-perception and request-versus-auto-pickup races, plus refusal without inventory evidence. Also handle a rejected vanished chest as an unmutated fallback, with permission rejection still terminal. Focused just check controller test-packages ./internal/behaviour passed. Gate/review/controller deploy and a new flint target follow; final live receipt will be attached to the follow-up PR. Background jobs remain uninstalled and check definitions are disabled between runs.

**agent:codex/renki-resume** at 2026-10-29T22:21:26Z

Resumed delivery on b325d9d plus the bounded home fallback, 22:14:31–22:15:14Z: collect revisited the consumed south-flint site with Flint2 already in the pack. Direct chest approach t-2 failed unreachable (3199 expansions/25593 probes; 674.12 ms total/9.07 ms worst frame). Stow then sent one move to the configured home for this chest within 10 m; t-3 eventually entered via the west door. Deposit t-4's matching receipt moved exactly Flint2, followed by newer inventory BeechSeeds15. Chest ended Stone1/Flint2/ArmorRagsChest1; HP remained 25/25. Evidence .checks/audit-window-LtoncZ.jsonl. The temporary definition was disabled afterward.

This is a resumed-delivery pass, not a fresh flint collection loop. It still tried underneath the base. The owner's observation led to a separate concrete navigation fix: height-aware waypoint advancement, with a replan instead of waiting directly underneath a waypoint on another floor. LocalRoute had already distinguished heights; PathFollower had discarded them using flat-only arrival. Saved entrance/successful-route reuse remains unimplemented. Full/warded storage, multi-chest live overflow, unattended repeated exits, leader resumption, deliberate owner-drop detection and default hauling remain open. Do not mark M2/hauling complete from these controlled loops.

**agent:codex/renki-resume** at 2026-10-29T22:26:17Z

Fresh one-target flint loop passed on the height-aware waypoint build with the bounded stow home fallback: 22:23:51–22:25:01Z, t-2 moved from home to (-28.15,30.43,42.11), t-3 used pickable 1:39745, t-4 took exactly Flint1, t-5's direct chest approach failed unreachable, t-6 went to configured home through the west door, and t-7 deposited Flint1. Task-matched receipt and newer pack observation confirmed chest Stone1/Flint3/ArmorRagsChest1 and pack BeechSeeds15. Every recorded state was 25/25 HP. No manual door staging occurred in this loop. Both temporary test definitions were deleted afterward. Evidence: .checks/audit-window-IM0XXq.jsonl.

This is a full bounded collection/delivery pass. It is not efficient/reliable direct chest routing: the direct approach still searched the underside and failed before the home fallback. Height-aware following prevents consuming waypoints from another floor; it does not add route memory. A reusable, live-validated entrance/successful-path hint is still needed, along with repeated unattended route checks. Tree cutting, mining, crafting, default loot, deliberate-drop detection, leader/work-site resumption and live multi-chest/full/ward checks remain open.
