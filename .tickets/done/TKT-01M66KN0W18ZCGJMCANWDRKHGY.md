---
schema: 4
id: TKT-01M66KN0W18ZCGJMCANWDRKHGY
title: Player control profiles, shared marker and Marco Polo
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - plugin
  - chat
  - protocol
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
created_at: 2026-10-30T14:33:04Z
updated_at: 2026-10-30T14:56:29Z
created_by:
  id: agent:codex/renki-presence
  name: ""
updated_by:
  id: agent:codex/renki-presence
  name: ""
extensions: {}
---

## Description

Implement the owner's requested shared map-position toggle and player-specific Marco/Polo routines. Design a common control-profile/permission model, preserving existing owners. Listed players joining out of sight trigger a shouted Marco. Authorized players shouting an addressed Marco receive a shouted Polo with straight-line direction/distance from Tas to the caller. A shouted Polo answering Tas's Marco receives directions from the speaker to Tas. Future route confidence/terrain reporting must distinguish known routes from straight-line estimates.

## Acceptance criteria

- [x] Validated player profiles govern chat commands and locate/greeting routines, preserving owners and task safety
- [x] Shared marker desired state persists; live on/off changes are reported by native telemetry without interrupting body control
- [x] Scoped joins and shouted Marco/Polo produce throttled replies with correct opposite direction semantics and no movement/model calls
- [x] Panel exposes marker, routine and profile controls; design documents route/terrain knowledge as a future capability
- [x] Replay tests, full CI and committed review pass; deploy and confirm marker state live

## Implementation plan

Add a central player-profile resolver with operator (commands/locate/join greeting), companion (locate/greeting), guest (locate), preserving existing owners as administrators/operators. Persist validated profile assignments and presence settings behind owner/panel authorization. Existing chat command authorization uses the resolver; routines check individual locate/greet permissions. A bounded, nonblocking presence manager consumes join/chat events with connection/world/freshness scope, delayed join visibility checks, throttling and a short Marco response window. Addressed shouted Marco replies Polo with bot-to-caller direction; shouted Polo during Tas's Marco window gives caller-to-bot directions. Neither triggers movement or an LLM. Add shared-position desired state to welcome/config and observed state to telemetry; plugin calls Valheim's ordinary SetPublicReferencePosition on Unity's thread, without disturbing body control. Add panel toggles/profile editor and docs for future route/terrain confidence. Test permissions, both bearings, visibility/grace, throttling, reconnect/world safety, and marker config; CI/review then deploy and live-toggle marker with observed confirmation.

## Notes

**agent:codex/renki-presence** at 2026-10-30T14:52:39Z

Implemented central operator/companion/guest profiles, owner precedence, persisted owner-only assignments, scoped/throttled presence routines, native marker telemetry/configuration and panel controls. docs/PLAYERS.md records permissions, opposite bearings, straight-line limits and future route/terrain confidence.

just check ci passed after the final scoped-dispatch fix: .checks/20261030T144941Z-BmCg1G.log (29 seconds). Includes race tests, native/core protocol fixtures and marker-check harness checks. Deterministic replay covers joins, visibility/grace, revocation, throttles, reconnect/world changes and both bearings. Scoped instant dispatch rechecks connection/world before publishing; replaced-connection chats are ignored. Panel preview with actual assets rendered roles/toggles/status without JavaScript errors. Marker hot configuration preserves control revision and automation in tests.

Remaining before completion: committed review, land/deploy, and native on/off/on marker verification. No human join/shout staging requested; that behavior currently has replay evidence rather than a live player exchange. M2/M3 melee/block/dodge evidence remains separate and incomplete.

**agent:codex/renki-presence** at 2026-10-30T14:56:04Z

Landed implementation in PR #73 (merge f4b8d6f, implementation b170d74). just check review passed with no findings (.checks/20261030T145240Z-0OVwwB.log). just check vm install passed with controller/package and installed DLL checksum verification (.checks/20261030T145338Z-Hubryw.log).

Live just check vm marker-check on passed in DizzysHellheim as Tasleroch: native sharePosition true, false, true, matching each desired transition. Final persisted desired state is true. HP remained at matching maximum (66.0 then 65.9), with no active task. Evidence: .checks/marker-check-20261030T145447768080Z/summary.json and .checks/20261030T145447Z-KULEHO.log. Automation/control-revision preservation is established by replay tests; this live check did not have an active body task.

No live multiplayer join/shout exchange was staged; those paths retain deterministic permission/bearing/freshness/reconnect/rate-limit coverage. Route confidence and intervening terrain remain designed future work; current responses explicitly report straight-line offsets.

## Summary

Implemented and deployed in PR #73: https://github.com/warricksothr/renki/pull/73 (merge f4b8d6f). Shared player profiles govern commands, locate and greeting routines; owners retain authority. Panel controls persist marker/routine settings and profile assignments. Both Marco/Polo bearings are covered by scoped, throttled deterministic replay. docs/PLAYERS.md documents future route/terrain confidence.

Full CI and committed review passed. Live marker on/off/on telemetry passed; sharing remains enabled. Tas was logged out and Valheim stopped after verification; controller/panel are running. Live multiplayer exchanges remain unstaged, and current directions remain straight-line. M2/M3 combat evidence remains tracked separately.
