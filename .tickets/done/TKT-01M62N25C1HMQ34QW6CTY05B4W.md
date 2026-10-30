---
schema: 4
id: TKT-01M62N25C1HMQ34QW6CTY05B4W
title: "Controller: danger zones from where the bot was hurt"
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - movement
assignees: []
milestone: M3
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies:
  - TKT-01M62EP0PDAB02DNJAJ6WS3MYD
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T01:40:46Z
updated_at: 2026-10-30T13:43:12Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

From memory's deaths and damage (TKT-01M62EP0P): places that hurt the bot become danger zones that decay over days. Paths and the choice of work site avoid them when there is another way, and the panel shows them.

## Acceptance criteria

- [x] Damage and deaths make danger zones that decay
- [x] Choosing a site or a route prefers to avoid them
- [x] The panel draws them

## Implementation plan

Persist bounded per-world damage/death observations and decaying danger zones beside existing memory. Capture precise plugin event position and associate only fresh current-connection events with a world; enqueue without disk IO on the bridge, retry batches with rollback, and flush at shutdown. Prefer safer initial gather and collect targets, expose current-world zones on the authenticated map, and render an optional danger layer. This slice changes target preference, not native navigation routes. Verify restart/decay/bounds/failed-save/world isolation/event freshness and target choice through replay fixtures, plus map scope and shutdown tests. Run focused just checks, full just check ci, and committed just check review before landing/deploying. Native combat and full gather/delivery remain separate live checks.

## Notes

**agent:codex/renki-resume** at 2026-10-30T13:36:08Z

Implemented persistent bounded per-world injury memory, decaying danger zones, safer initial gather/collect target ranking, and the optional Danger map layer. Plugin injury/death events now include exact position and attacker prefab when available; unknown causes stay unknown. Session snapshots event scope atomically with current-connection/fresh-state/time/sequence checks, so delayed events cannot be attributed to a replacement world. The bridge only queues bounded observations; one-second worker batches, retries with rollback/original times, and graceful shutdown drains pending memory. Store retains 256 injuries/world for seven real days, derives at most 128 twelve-metre zones, separates heights, and halves each injury contribution daily.

Focused just checks passed for store restart/decay/bounds/invalid batches/rollback, recorder retries/world/time/overflow/shutdown, fresh connection/event scoping including zero-HP death states, engine memory wiring, safe tree/pickable ranking with decay, and current-world map isolation. just check ci passed (controller race suite, builds, map tests, plugin build/core suite, ticket store, supervisors). Browser preview using actual panel assets showed one fixture danger zone, injury/attacker/time tooltip, working layer off/on, and no page errors. This implements the site/target-preference branch of AC 2; explicit site travel, delivery, return-home and native route avoidance remain unchanged, as documented in docs/MEMORY.md. Missing-object inference on the broader M2 memory dependency remains open because scan omission is not complete-coverage evidence. No intentional live damage/death was required for this memory slice.

Remaining before landing: committed review, PR merge, verified installation. M2 full gather/delivery and positive M3 melee/block/dodge/pursuit still require live evidence.

## Summary

Implemented and merged in PR #68 (8421e67), installed with checksum verification in runtime 552c491. Per-world bounded injury/death memory drives daily-decaying danger zones, safer initial gather/collect selection and the optional map Danger layer. Fresh atomic connection/world/event scoping, retry/rollback/shutdown persistence, bounds/decay and target preference have passing regressions; full CI, committed review and real-assets browser checks passed. Explicit travel/delivery/home/native route avoidance and broader milestone live combat/gathering checks remain separate.
