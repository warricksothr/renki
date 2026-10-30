---
schema: 4
id: TKT-01M66N83DFDHKKXXJ7GQTK4RES
title: Remember player encounters and answer last-known locations
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T15:00:58Z
updated_at: 2026-10-30T15:17:59Z
created_by:
  id: agent:codex/renki-player-memory
  name: ""
updated_by:
  id: agent:codex/renki-player-memory
  name: ""
extensions: {}
---

## Description

Remember players Tas sees or hears, including ordinary chat and Marco/Polo. Keep bounded per-world encounter histories with progressively coarser location/time resolution and consolidation as observations age. Deterministic addressed location questions should report last seen/heard time and straight-line offset, using the requesting player's recent current-session position when known and otherwise explicitly using Tas. Reuse locate permission; no movement, model calls or inferred routes.

## Acceptance criteria

- [x] Persist bounded world-scoped seen/heard histories with age-based precision loss, consolidation and expiry
- [x] Capture all located speech and fresh perception without blocking the bridge or refreshing stale sightings
- [x] Authorized location questions name age/source and use a recent current-session requester position or explicit Tas fallback
- [x] Tests cover persistence/aging, scope/freshness, bearings/reference fallback and permission/model/task safety; document and deploy

## Implementation plan

Extend shared memory.json with per-world player observations, source and explicit spatial/time resolution. Consolidate to nested 5/25/100/500 m cells and 1 s/1 min/15 min/1 h time buckets as age crosses 10 min/6 h/2 days; expire at 14 days. Bound worlds, players and per-player history. Reads age immediately; 30-second persistence sweeps physically discard precision. Failed saves roll back store mutations and retain pending events.

Capture heard positions on the session read goroutine with atomic current-connection/world/fresh-state scope; exclude pings/panel and invalid coordinates. A nonblocking memory recorder samples new fresh perception and batches writes. Keep separate current-session recent requester positions (30 s maximum) that do not revive from disk/reconnects. Reuse presence's bounded deterministic reply queue and locate capability for addressed where-is/last-seen queries, with ordinary/shout/whisper replies and panel support. Report last observation age/source, uncertainty, and explicit reference point; never claim current target or route. Test layers and app wiring, document, CI/review/land then controller deploy and read-only verification.

## Notes

**agent:codex/renki-player-memory** at 2026-10-30T15:13:03Z

Implemented bounded player histories in shared memory.json, aged nested spatial/time buckets, age-tier sampling preserving older representatives, expiry/world/player/history caps and atomic save rollback. Recorder captures all located normal/whisper/shout chat on the session read goroutine with atomic scope/freshness/sequence validation; fresh new perception is sampled once a second. Persistence/aging sweeps run every 30 seconds and flush on shutdown, retaining failed batches and deferring observations newer than a sweep's timestamp. Current requester references are separate, expire at 30 seconds and never reload from disk or another connection.

Deterministic addressed where-is/last-seen queries reuse locate permission and presence queue/throttles, reply in the incoming channel and do not depend on Marco/Polo being enabled. The panel command box supports the same queries without fabricating a player position. Replies name source, age, remembered-area uncertainty and requester/Tas reference explicitly; no current-location or route claim. Authenticated /api/memory/players exposes aged bounded history. docs/PLAYERS.md and docs/MEMORY.md describe thresholds and limits.

just check ci passed in 29 seconds: .checks/20261030T151135Z-Cg5rIA.log. Includes race tests, history persistence/rollback/fading/consolidation/limits, stale/replayed/unlocated speech, fresh-frame world scope, requester-reference expiry/reconnect, opposite reference bearings, permissions/revocation/throttles, authenticated API and app query bypass of an enabled fake voice without control revision changes.

Remaining: committed review, land/deploy, and a brief live smoke check of the memory API and deterministic panel query. No imported audit history and no multiplayer staging requirement; real new encounter capture will depend on players being present/heard.

**agent:codex/renki-player-memory** at 2026-10-30T15:17:59Z

Implementation landed in PR #75: https://github.com/warricksothr/renki/pull/75 (merge fdae84a, implementation b9f99d7). just check review passed with no findings (.checks/20261030T151303Z-O9MGzQ.log). just check vm install passed with verified package/installed DLL checksums (.checks/20261030T151446Z-Jitr2G.log).

Live smoke ran the deployed fdae84a in DizzysHellheim. Authenticated /api/memory/players returned matching world and zero encounters. Panel question from Nimah, "where is Sothr?", produced "I have no remembered location for Sothr in this world." Audit confirms no native task in the query window. HP 65.3/65.3, no body task, marker still shared. Evidence: .checks/player-memory-live-summary.json and the audit file named by .checks/player-memory-live-audit-path.txt. The first inspection script expected log fields at the top level; corrected inspection of attrs confirmed the reply from the same retained audit without replaying the action.

No positive live player encounter was observed/staged. Seen/heard capture, persistence/fading, requester reference and positive replies have deterministic replay coverage; this smoke confirms deployed API/query wiring and honest unknown behavior. No audit backfill was performed. Tas logged out, Valheim stopped, controller/panel running after cleanup.

## Summary

Landed/deployed in PR #75 (fdae84a). Durable per-world seen/heard player histories now consolidate, lose spatial/time precision with age and expire after fourteen days. Every located speech channel, including Marco/Polo, and fresh player perception feed bounded memory. Authorized deterministic location questions report last evidence/source/age/uncertainty relative to a current-session requester position no older than thirty seconds, or explicitly Tas. Read-only authenticated history API and documentation included.

Full CI and committed review passed. Live API and unknown-player panel query passed with no native task, healthy Tas and marker sharing preserved. Positive encounter/fading/reference paths have replay coverage; no live multiplayer encounter was staged. Tas is logged out and the game stopped; panel remains running.
