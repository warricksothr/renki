---
schema: 4
id: TKT-01M62EP0PDAB02DNJAJ6WS3MYD
title: "Controller: remember where things are and should be, per world"
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62BMD18K3SE1WXJ7FDN8EPT
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-resume
  branch: t3/chest-memory
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: e0a7f40715409b6a1107bac325c49aa2f0b40a48
  session: null
  claimed_at: 2026-10-29T20:14:14Z
  expires_at: null
archive: null
created_at: 2026-10-28T23:49:16Z
updated_at: 2026-10-29T20:18:38Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

The bot forgets everything it sees. Perception (TKT-01M62BMD1) tells the
controller what is around the bot now; memory keeps it, per world, so the
bot knows where things are when they are out of sight, and notices when
they are not where they should be.

### What it remembers

- **Places and things:** structures from perception (beds, chests,
  stations, portals, doors, fires) with position, prefab and when last
  seen; resources worth coming back to (ore, berry bushes); the home.
- **Expectations:** a remembered thing that perception should see from
  where the bot stands but does not is marked missing, with when; one that
  moved is updated. That is what "should be" means: the last known state,
  checked against the present.
- **Deaths and danger:** each death with place, cause (attacker id or
  prefab, drowning, falling) and time; places that hurt the bot become
  danger zones that decay over days, for pathing to avoid and the panel to
  show.

### Where it lives

The controller, beside the homes in dataDir (or its own file there),
written as perception arrives and pruned as entries age. The panel map
draws remembered things faintly and current ones solidly. The LLM summary
can say what is near ("there's a chest by the workbench").

## Acceptance criteria

- [ ] Structures and notable resources from perception are kept per world across restarts, with when each was last seen
- [ ] A remembered thing in view that perception no longer reports is marked missing
- [ ] Each death is recorded with place, cause and time, and makes a danger zone that decays
- [ ] The panel map draws remembered things and danger zones

## Implementation plan

Continue the memory step already selected in the owner's M2 roadmap, beginning with the exact chest observations needed by stow. The deployed perception and container interfaces already exist; their remaining live visual/capacity/ward checks remain open and are not prerequisites for implementing persistence against those interfaces.

Add bounded per-world remembered containers to memory.json beside homes/graves, retaining entity, position, observed prefab/kind when known, size, exact items and wall-clock observation time. Replace whole observations, reuse Inventory.Room, return independent copies with explicit age/freshness and roll back failed saves. Batch current-session chest observations from an atomic session/world snapshot once per second, without sending game commands, and retry failed persistence without stamping old data as newly observed.

Feed fresh remembered stock into behaviour conditions after controller restarts, with current live observations taking precedence, and expose remembered contents/age through an authenticated read-only API. Test world isolation, restart, replacement, room, ages, rollback, reconnection and live-over-memory precedence. Run focused just checks, just check ci and committed just check review before landing and controller-only deployment. Keep the broader memory ticket in progress: notable resources, missing-object inference, death/danger zones and map rendering are outside this chest slice. Live route and combat checks stay deferred.

## Notes

**agent:claude-code/renki-groom** at 2026-10-29T01:16:12Z

### Fidelity and attention (from Nimah, 2026-10-28)

The bot should not remember everything at the same fidelity. It should save its attention for the values that change and can drive behaviour. The design, given that:

- **Flyweights.** Each prefab is described once, in a per-world catalogue: kind, minTool, what it yields, and whether it is a material worth collecting. A remembered thing holds only its id, an index into that catalogue, a position rounded to 0.5 m, when it was last seen, and the few fields that change. A forest of thousands of trees then costs a few bytes a tree.
- **Fidelity tiers.**
  - **Scenery** (trees, rocks, logs): it never moves, so it is kept as counts per map cell ("cell (-40, 0): 14 Beech1, 2 rock4_coast_frac"), not one record each. The exceptions are things worth coming back for (ore, berry bushes), which are kept individually.
  - **Structures:** one record each. They rarely change; a door's open state is transient and is not stored.
  - **High signal:** drops, creatures and players, tracked individually with a short history. Drops are grouped per cell and prefab, with the total stack and its trend over time.
- **Attention is change.** The controller derives signals from differences, not from snapshots. Examples:
  - "Wood on the ground near (x, z): 12, up from 3 a minute ago" (chopping leaves a pile).
  - "Stone and copper ore accumulating by the mine."
  - "A creature has the bot as its target."
  - "A chest that should be here is not."

  Signals are what the brain, the LLM summary and future behaviours act on. For example, picking up a growing pile when the inventory has room needs the inventory stream (6.3) and a pickup task (M2, behind AllowGather). That behaviour is for M2 grooming.
- **The wire can follow suit later.** Characters and drops matter on every scan; scenery and structures could go out only when they change. The v1 snapshot is about 25 items a second, so this can wait until size matters (6.2 already says deltas come later).

**agent:codex/renki-resume** at 2026-10-29T20:18:38Z

First per-world memory slice implemented on t3/chest-memory: exact container observations now persist beside homes/graves in memory.json, with entity, position, kind/prefab when known, size/items and the wall-clock arrival time. Session captures that time on receipt and returns atomic connection/world observation snapshots. A once-per-second recorder saves only newly received frames, replaces whole contents (including empty), retries failed saves without changing observation age, and scopes tracking to each connection/world. Store reads return copies; room reuses Inventory.Room; retention is bounded to 4096 records per world with age/oldest pruning on updates, and failed writes roll back memory.

Behaviour stock conditions can use remembered contents after restart, retaining explicit maximum ages and refusing future-dated data. Live contents always supersede a remembered duplicate, including an empty chest. Authenticated GET /api/memory/containers exposes current-world contents with observation time and ageS. docs/MEMORY.md describes freshness, room limits and retention.

Focused just check controller test-packages checks and just check ci passed, including the race suite, Linux/Windows builds, plugin build and 158 C# tests. New regressions cover restart, whole-snapshot replacement, input/output isolation, room, age/future dates, world separation, bounded retention, failed-save rollback, recorder retries preserving arrival time, stale-world exclusion, live-over-memory stock precedence and read-only API authentication/age output.

The owner previously selected the M2 roadmap and requested autonomous continuation; this promotes its memory step. Perception and transfer interfaces are already deployed, so their outstanding visual/capacity/ward live checks do not prevent developing this persistence slice. Those checks remain open. The larger ticket is intentionally in progress with its acceptance boxes unchanged: notable resources, missing-object inference, death causes/danger zones and map rendering remain unimplemented. The chest-memory ticket TKT-01M62VXYWJHB5BC9VGCBM0PVXD (Remember what each chest holds, its size and when it was seen) remains draft and is not declared complete from this narrower slice. Live route/combat checks stay deferred. Committed review, landing and controller deployment follow.
