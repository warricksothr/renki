---
schema: 4
id: TKT-01M62BMD18K3SE1WXJ7FDN8EPT
title: "Plugin: send perception of characters, structures and resources"
type: task
status: review
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - protocol
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
  branch: t3/perception
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 256aac72646759b8fd99fe016739557d7e7be88a
  session: null
  claimed_at: 2026-10-29T00:56:54Z
  expires_at: null
archive: null
created_at: 2026-10-28T22:55:57Z
updated_at: 2026-10-29T02:09:28Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The protocol defines `perception` (docs/BRIDGE-PROTOCOL.md 6.2), but the
plugin has never sent it: TKT-01M61T36M4 left perception for M2. Without it
the controller and the panel know nothing around the bot but chat, and the
panel's map shows only the bot itself.

### What to send

At `perceptionHz` (default 1), within `radius`, nearest first, capped at
`maxPerKind` per kind, as 6.2 describes:

- `characters`: `Character.GetCharactersInRange`; players by name, creatures
  by prefab, with hpFrac, stars, hostile, targetingMe, and tamed.
- `structures`: `Piece.GetAllPiecesInRadius`, reduced to the kinds 6.2 names
  (container, station, bed, portal, door, fire, smelter). Doors carry
  whether they are open, which 6.2 does not have yet: add it there.
- `resources`: trees, logs, rocks, pickables and drops, by component
  (TreeBase, TreeLog, MineRock, Pickable, ItemDrop).

### Uses

- The panel's map draws what is really around the bot.
- The controller and the LLM's summary know about doors, beds, portals and
  threats.
- TKT-01M62ACH8 (sparse occupancy grid spike) can rule doors in or out from
  real data.

### Watch

Frame size and frame time: a base can hold hundreds of pieces, so the caps
matter, and the scan should not run every physics frame.

## Acceptance criteria

- [x] The plugin sends perception at perceptionHz with characters, structures and resources as 6.2 describes, capped and nearest first
- [x] Doors report whether they are open; 6.2 and the fixtures on both sides say so
- [ ] The panel's map shows players, creatures and structures from the real game
- [x] A perception scan in a built-up base takes under 2 ms, measured in the game

## Implementation plan

- **Protocol:** structures gain `open` (doors only) in 6.2, the Go and C# types, and the shared fixture.
- **Renki.Core:** `Nearest<T>` keeps the nearest N items of a kind, tested without the game.
- **Plugin, Perceiver.cs:** scans from Update (not physics) at perceptionHz from welcome or config, while in a session and in the world.
  - **Characters:** `Character.GetCharactersInRange`, excluding the bot itself. Kind is player, tamed or creature. Players carry their name; creatures carry prefab, stars (`GetLevel()-1`), hostile (`BaseAI.IsEnemy` against the bot) and targetingMe (`GetBaseAI().GetTargetCreature()`).
  - **Structures:** `Piece.GetAllPiecesInRadius`, keeping only these components: Door (open from the ZDO's `s_state`), Bed (`mine` from `IsMine`), TeleportWorld (tag from `GetText`), CraftingStation, Smelter, Fireplace and Container. Walls and floors are left out, so they cannot fill the cap.
  - **Resources:** `Physics.OverlapSphereNonAlloc` on the layers that hold trees, rocks, pickables and drops, de-duplicated by ZNetView. Components: TreeBase (tree), TreeLog (log), MineRock and MineRock5 (rock, with `minTool`), Pickable not yet picked (pickable), ItemDrop (drop, with stack).
  - **IDs:** the ZDOID, the same `user:id` form the events already use.
  - **Timing:** a Stopwatch around each scan, with a trace every 30 s carrying the counts and the average and worst ms, so the 2 ms budget can be read from the audit log.
- **Controller:** the session keeps the newest perception (for the panel status and, later, memory). The panel map draws structures by kind and creatures coloured by hostility, and shows door state.

## Notes

**agent:claude-code/renki-groom** at 2026-10-29T01:07:49Z

Measured in the game on 2026-10-28 at home in DizzysHellheim, with 317 pieces, 105 colliders, 9 structures and 15 resources in range, sending at 1 Hz.

- **First build:** 1.75 ms on average and 2.31 ms at worst (20:03:19), over budget at worst. Most of it went to checking seven component types on every piece, every scan.
- **With what each piece and object is cached:** 0.73 ms on average, 0.86 ms at worst (20:06:41); then 0.80 ms and 1.02 ms (20:07:11).
- **The first scan after a start** takes about 15 ms, while the game warms up and the caches fill. It happens once.

The plugin sends this as a `perception: scan` trace every 30 s, so `just vm audit 1 | grep 'perception: scan'` shows it in any base.
