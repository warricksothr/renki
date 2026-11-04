---
schema: 4
id: TKT-01M6KZ8KGV2MQANWADECVXBSPP
title: Ask players about bedtime once per game night
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - chat
  - plugin
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/explore-night
  branch: t3/renki-portal-travel-startup
  worktree: /home/sothr/.t3/worktrees/renki/t3-6c0da0c0
  commit: f0c4869dc65a0dbec99519423b2386c362b35ada
  session: null
  claimed_at: 2026-11-04T19:06:22Z
  expires_at: null
archive: null
created_at: 2026-11-04T19:06:08Z
updated_at: 2026-11-04T19:15:27Z
created_by:
  id: agent:codex/explore-night
  name: ""
updated_by:
  id: agent:codex/explore-night
  name: ""
extensions: {}
---

## Description

The owner requests a night routine that asks people whether they want to go to bed, at most once per game day/night. Add an explicit persistent setting in Players & presence and a deterministic invitation alongside existing presence routines. Preserve current tasks; asking must not put Tas or other players to bed.

Use native game night/day observations and a durable world/character high-water mark so reconnects, controller restarts, midnight and backwards clock changes cannot repeat the same night's invitation. Missing/stale time or audience telemetry must not invent an audience. Support local normal chat and shout, with the owner selecting the initial mode. Save the attempt before sending to avoid replay after uncertain delivery.

## Acceptance criteria

- [ ] Night invitations use fresh native time and an observed audience, preserve body tasks, and never start sleep automatically.
- [ ] A durable world/character limit prevents repeats across midnight, reconnect, restart, send uncertainty and clock rollback; a later night can invite again.
- [ ] Players & presence exposes a persisted off/nearby/shout setting; meaningful regressions and full CI pass.
- [ ] Deploy and enable the selected mode, retain Tas/private output online, and document any live audience/night check that remains unearned.

## Implementation plan

Extend existing Presence routines with a persisted presence.bedtime mode: off(empty), normal(local chat), or shout. The panel exposes the same choices. Native state gains a nullable count of other server players, sampled from the actual player list, so shout mode does not invent an audience; local mode uses fresh scoped perception and ordinary chat range.

Before an instant scoped say order, reserve that night's invitation in home memory, keyed by native world UID and Tas's character. Validate native night/day/fraction and scope; map early-morning night to the previous evening so midnight is not another invitation. Save a high-water mark before dispatch, rollback on persistence failure, never evict saved marks to regain permission, and do not replay uncertain delivery. Skip stale/offline/sleeping/alone states and preserve existing body control. This only asks; it does not start sleep or treat replies as automatic consent.

Cover persistence/restart, midnight, rollover, backwards time, distinct worlds/characters, write failure, absent audiences/time, disabled mode, reconnect, uncertain send and active body control. Run focused checks, full CI, commit/review, deploy the real plugin/controller and enable the owner's selected mode. Keep any real-player/night delivery check explicitly unearned if the required scene is absent.

## Notes

**agent:codex/explore-night** at 2026-11-04T19:15:27Z

Owner selected shout when other players are online. Implemented optional native server-roster count, durable world/character calendar-day and overnight attempt ledger, and persisted presence.bedtime modes. Initial CI caught stale generated panel JS; just panel build regenerated it. A separate focused app run hit the pre-existing retreat fixture timeout, then ten focused race repetitions passed; production safety guards were unchanged. Complete CI is running.
