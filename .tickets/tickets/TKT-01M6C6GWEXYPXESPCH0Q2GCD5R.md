---
schema: 4
id: TKT-01M6C6GWEXYPXESPCH0Q2GCD5R
title: Remember pickup origins and stow surplus cargo after loops
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - plugin
  - safety
assignees: []
milestone: M2
parent: TKT-01M62N12HJW32JJTD2M3PEAEA4
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-milestones
  branch: t3/cargo-memory
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: ec7ab7188c941c8c379517a2304430d61e2aa8d2
  session: null
  claimed_at: 2026-11-01T18:41:09Z
  expires_at: null
archive: null
created_at: 2026-11-01T18:39:04Z
updated_at: 2026-11-01T19:27:10Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Tas finishes loops with incidental copper and coal still in his pack. Record where actual ground acquisitions occur, with world/session/item/count/time and nearby chest candidates, and use exact last-known chest contents as storage clues. At a safe storage-marker idle boundary after loops, inspect candidates near the pickup and deposit surplus into chests already holding the same item. Empty/unknown chests are discovery candidates, not automatic assignments. Keep equipped gear, tools, food reserves and explicitly reserved cargo; unknown/full/busy/missing destinations retain cargo and report the shortage. Preserve newer orders and native physical/combat guards. Use existing stow/transfer receipt machinery and shared durable memory rather than a new script engine.

The tutorial raven has also obstructed Tas. Suppress tutorial raven spawns for this local bot through a configurable plugin setting; retain ordinary non-tutorial interactions and avoid server/player changes.

## Acceptance criteria

- [ ] Actual ground pickups retain bounded durable world/item/location/time provenance across restart
- [ ] Idle storage work inspects nearby candidate chests and stows only into an observed matching destination
- [ ] Surplus stow preserves gear, food and configured reserves and yields to newer commands
- [ ] Tutorial raven spawns are disabled for Tas through a configurable local plugin setting

## Implementation plan

Observe successful local Humanoid.Pickup calls with before/after pack counts, original ground position/entity and prefab. Send a typed item_picked_up event only for positive actual gain. Scope and deduplicate observations against fresh current connection/world state, retain a bounded session ring, and persist bounded per-world pickup records through the existing memory recorder. Expose origin hints to the behaviour view and shared container-memory API.

Add marker_cargo on the existing idle engine: fresh safe storage-marker admission, protected food/gear, configured keep counts, finite cargo/chest/time budgets and normal command cancellation. Rank unique nearby chest candidates by known matching contents and then proximity to recent pickup origins; also inspect unknown chests within the storage area. Use existing stow receipts for one prefab at a time, adding a deposit-only matchExisting option so the native transfer opens/observes the chest and moves nothing unless it currently contains that prefab. This closes the inspect/deposit race and prevents empty/wrong chests from becoming assignments. Exhausted/busy/full/unknown destinations retain cargo and report a storage need. Enable one home-storage idle definition after guarded deployment/readiness; do not interfere with a newer human command.

Suppress tutorial Raven.Spawn only for tutorial text on the local plugin through Gameplay.SuppressTutorials, configurable and on by default for the bot. No change to other clients, world collision, or non-tutorial text. Validate focused controller/session/memory/behaviour/protocol and plugin checks, full CI, committed review, stacked PR and installation. Record east-meadow hostile refusal and leave broader route/pantry acceptance open; no unchanged route replay.

Live home unload stored Copper11, BeechSeeds9, Feathers2, Resin7 and Wood25, retaining Wood10 and Coal45. Exact matched receipts and newer pack frames earned the deposits; unknown/mismatched/full candidates moved nothing. A Coal1 drop/pickup produced durable scoped origin evidence that survived a controller-only restart. Local saved plugin configuration confirms SuppressTutorials=true. Two currently perceived exterior chests near the production drops (1:73618/1:73622) could not be reached: validated porch prefix then endpoint_support, no hostile rejection. Tas returned inside uninjured.

Before concluding exterior storage is inaccessible, share cooking's bounded physical interaction approach with conditional chest transfers. After nearby unreachable/stuck only, measure supported, dry, body-clear standing points with actual eye reach and line of sight; use ordinary dry/hostile-avoiding move_to, at most144 seeds/four walks/60s. Never retry water/permission/stopped refusals or alter chest access and exact transfer verification. Preserve cooking heat/fuel checks through a wrapper. Cover fallback eligibility and shared candidate geometry, compile, full gate, committed review, publish/deploy and retry the two exterior conditional destinations under normal guards. Record safe refusal honestly if they still cannot be used.

Native interaction recovery succeeded on7f39fc7: first exterior chest opened and refused Coal because it contains CopperOre/TinOre; second chest accepted all45 Coal with a matched native receipt. Tas auto-picked three new Copper ingots at the production drops; native pickup events retained their real source locations, and he returned safely inside. Add a separate observed home workshop storage marker without moving home. Extend marker_cargo to visit only explicit storage markers/chests within40m of admission, pin source/destination identity, preserve strict initial floor admission and common health/food/threat/injury/time guards, and verify a return to the source floor before reporting completion or shortages. Conditional native transfers own their complete safe approach; bypass the controller move_to at solid chest centers. Deterministic checks cover unnamed/remote floors, edited markers, threats/injury, exact deposit receipts and state-backed home arrival. Then deploy controller-only and prove an automatic home-to-workshop Coal1 deposit and return; keep all material conservation and player-order guards.

## Notes

**agent:codex/renki-milestones** at 2026-11-01T18:50:36Z

Implemented native ground-pickup events with source position/entity and measured positive pack gain (including partial auto-pickups). Generic inventory changes and chest withdrawals do not create origin records. Session ingress validates fresh current-world UID/connection, timing, sequence/count/position and keeps a256-record ring. Existing shared-memory worker persists at most256 records/world for14 days with stable arrival IDs, rollback and retry deduplication. Shutdown awaits its final flush. Container-memory API exposes these records.

Implemented marker_cargo using existing30s idle admission, five-minute cooldown,100-run bound, six-minute native task guard and cancellation. It protects all consumables, gear and unknown types; preset keeps10 Wood, custom keep maps remain configurable. It ranks known matching live chest IDs, then recent scoped pickup proximity, then the other unique currently seen chests within storage marker/floor. It tries each once per material; tombstones and missing/other-floor IDs are excluded. A single-prefab stow reuses native receipts and newer pack checks. Deposit-only matchExisting opens under normal access/ownership and refuses without moving anything if the current chest lacks the material. The controller requires the new hello feature, so older plugins cannot silently ignore the argument. Full/busy/unmatched results preserve cargo and report a bounded storage need.

Gameplay.SuppressTutorials defaults true on the local bot plugin and gates tutorial Raven.Spawn only. Non-tutorial texts and other clients/server settings are unchanged. New plugin build passes. Focused controller/session/home/memory/behaviour/app checks and TransferTests pass; full CI and committed review follow. No native cargo transfer, pickup/restart or tutorial-spawn proof yet.

Route regression remains distinct: PR150 published with full CI and committed review passed. Storage and south-meadow round trips passed without injury. East-meadow .checks/routes-20261101T183424910659Z refused hostile_corridor beside BossStone_Bonemass/stone after109.2m sampled travel; no complete trip. Supervisor safely logged out at[8.39,33.28,-4.5],62.8/62.8HP and restored the panel. Last visible Greyling was36.5m away toward the failed east objective. Do not retry that objective unchanged; a guarded local return toward home is a different route and must retain all existing native health/food/threat admission checks before any storage trial.
