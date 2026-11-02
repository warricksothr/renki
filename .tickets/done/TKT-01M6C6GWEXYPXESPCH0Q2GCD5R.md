---
schema: 4
id: TKT-01M6C6GWEXYPXESPCH0Q2GCD5R
title: Remember pickup origins and stow surplus cargo after loops
type: task
status: done
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
claim: null
archive: null
created_at: 2026-11-01T18:39:04Z
updated_at: 2026-11-01T19:35:42Z
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

- [x] Actual ground pickups retain bounded durable world/item/location/time provenance across restart
- [x] Idle storage work inspects nearby candidate chests and stows only into an observed matching destination
- [x] Surplus stow preserves gear, food and configured reserves and yields to newer commands
- [x] Tutorial raven spawns are disabled for Tas through a configurable local plugin setting

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

**agent:codex/renki-milestones** at 2026-11-01T19:35:41Z

Implemented, published in PR151 and deployed controller a1c80b2 with plugin7f39fc7. Full just check ci passed172s (.checks/20261101T192749Z-WdH6HN.log); committed just check review 7f39fc7..HEAD passed with no findings (.checks/20261101T193057Z-qk79Jc.log). The earlier shared interaction build also passed CI163s and committed review with no findings. Checksum-verified installation and controller-only deployment retained game state as intended.

Native automatic home unload stored Copper11, BeechSeeds9, Feathers2, Resin7 and Wood25, keeping Wood10, gear and food. Matched container receipts/newer pack observations earned each move. Ten unique home-floor chests did not currently match Coal, so Coal45 was retained. A native Coal1 drop/pickup generated measured source position/entity, world UID, count and time; the identical durable origin survived controller and game restarts. Three real Copper pickups at production outputs also produced source records. Local saved plugin configuration confirms SuppressTutorials=true; tutorial-only Harmony suppression is installed. This does not attribute the earlier east-meadow hostile corridor or every obstacle to the raven.

Initial exterior routes to1:73618/1:73622 refused endpoint_support after a safe porch prefix. Shared measured interaction approaches subsequently reached both: the first opened with copper/tin ore and refused Coal without movement; the second accepted all45 Coal. A separate observed home workshop storage marker now identifies that floor without editing home or home storage. Cargo may visit explicit nearby storage markers within40m, pins both marker identities, retains original-floor admission and common physical/health/food/threat/injury/lease/time guards, and verifies its return before reporting completion or shortages. Conditional transfers own their full native approach instead of an earlier move_to at a chest center. Deterministic checks cover marker edits, unnamed/remote floors, threat/injury/local bounds, exact receipts and return despite rejected cargo.

Automatic journey acceptance passed after controller-only deployment, using the normal30s idle dwell (no manual behaviour start). Exact two tasks: t-1 conditional deposit Coal1 into1:73622, matched receipt/newer pack; t-2 dry/hostile-avoiding return to home storage. The approach tried a second measured clear standing position after its first12s walk budget. Fresh grounded shelter at[-39.57,36.09,8.02] proved arrival; minimum70.3HP, injurySeq0 throughout. Final pack: equipped rags/axe, CookedDeerMeat1, Mushroom1, Wood10, no Copper or Coal. Home-storage cargo remains enabled with default reserves, five-minute cooldown and100 runs/session. Evidence .checks/cargo-20261101T1900Z/summary.json and .checks/audit-window-ibzxah.jsonl; direct/home receipt audit .checks/audit-window-jYhjca.jsonl.

A startup RETURN profile initially refused one active meal and logged out as designed. Tas was already sheltered at home; ordinary startup retained its guard, then existing food keeper replenished two active meals from observed chest withdrawals. No thresholds were lowered and no manual rescue/staging was needed. Broader M2/M3 route, coastal detour, ten-return reliability and pantry production acceptance remain open on their own tickets.

## Summary

Deployed durable ground-pickup provenance, conditional surplus storage and configurable local tutorial suppression through PR151. Live home sorting stored the initial copper and other materials; measured interaction recovery reached outdoor storage and deposited all45 Coal. A separately staged Coal1 then proved the complete automatic30s-idle home/workshop deposit/verified-return journey, with exact native receipts, no observed injury and preserved reserves. Pickup memory survived controller and game restarts. Tas is idle, healthy and sheltered at home; normal cargo idle automation stays enabled. Full CI and committed reviews passed. Broader M2/M3 route/pantry criteria remain open.
