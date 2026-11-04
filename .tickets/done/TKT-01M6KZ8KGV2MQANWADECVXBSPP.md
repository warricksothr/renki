---
schema: 4
id: TKT-01M6KZ8KGV2MQANWADECVXBSPP
title: Ask players about bedtime once per game night
type: task
status: done
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
claim: null
archive: null
created_at: 2026-11-04T19:06:08Z
updated_at: 2026-11-04T20:09:34Z
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

- [x] Night invitations use fresh native time and an observed audience, preserve body tasks, and never start sleep automatically.
- [x] A durable world/character limit prevents repeats across midnight, reconnect, restart, send uncertainty and clock rollback; a later night can invite again.
- [x] Players & presence exposes a persisted off/nearby/shout setting; meaningful regressions and full CI pass.
- [x] Deploy and enable the selected mode, retain Tas/private output online, and document any live audience/night check that remains unearned.

## Implementation plan

Extend existing Presence routines with a persisted presence.bedtime mode: off(empty), normal(local chat), or shout. The panel exposes the same choices. Native state gains a nullable count of other server players, sampled from the actual player list, so shout mode does not invent an audience; local mode uses fresh scoped perception and ordinary chat range.

Before an instant scoped say order, reserve that night's invitation in home memory, keyed by native world UID and Tas's character. Validate native night/day/fraction and scope; map early-morning night to the previous evening so midnight is not another invitation. Save a high-water mark before dispatch, rollback on persistence failure, never evict saved marks to regain permission, and do not replay uncertain delivery. Skip stale/offline/sleeping/alone states and preserve existing body control. This only asks; it does not start sleep or treat replies as automatic consent.

Cover persistence/restart, midnight, rollover, backwards time, distinct worlds/characters, write failure, absent audiences/time, disabled mode, reconnect, uncertain send and active body control. Run focused checks, full CI, commit/review, deploy the real plugin/controller and enable the owner's selected mode. Keep any real-player/night delivery check explicitly unearned if the required scene is absent.

## Notes

**agent:codex/explore-night** at 2026-11-04T19:15:27Z

Owner selected shout when other players are online. Implemented optional native server-roster count, durable world/character calendar-day and overnight attempt ledger, and persisted presence.bedtime modes. Initial CI caught stale generated panel JS; just panel build regenerated it. A separate focused app run hit the pre-existing retreat fixture timeout, then ten focused race repetitions passed; production safety guards were unchanged. Complete CI is running.

**agent:codex/explore-night** at 2026-11-04T19:35:58Z

Deployed commit4e57efd with just check vm install and just check vm online; preflight/checksums/receipt passed and private output restored. Enabled persisted presence.bedtime=shout. Native state reported otherPlayers=1. On native day213 at night onset, actual t-13 sent exactly “It's night. Does anyone want to go to bed?” via shout, accepted/succeeded at19:26:21Z. Exact audit .checks/audit-window-sPuxk5.jsonl and deployed status are retained. Tas remained awake; no sleep task was dispatched. Native state later crossed midnight to day214/night without a repeat observed so far; final deployment will verify restart deduplication against the same durable reservation. FullCI for4e57efd passed220s and committed review was clean. The later food batch adds correlated successful-attempt logging; it does not reset the invitation ledger.

**agent:codex/explore-night** at 2026-11-04T19:54:59Z

This supersedes the interim no-repeat inference in the19:35 note. Complete exact audit19:24:00Z..19:45:00Z (.checks/audit-window-nxUeVD.jsonl) contains two bedtime shouts: t-13 at19:26:21Z and t-20 at19:30:50Z. Native day advanced213→214 while smoothed lighting dayFraction still read0.998. GetDayFraction is delayed lighting, while GetDay reads the raw network clock; combining them misidentifies the same night. No automatic sleep was sent.

Corrected native telemetry now samples calendar day and calendarFraction coherently from one native clock reading and configured day length. The routine requires this raw optional field, uses it for night identity and requires raw night plus native night; older/missing clocks defer. Added the exact live midnight mismatch as a presence regression plus pure native clock tests and time-skip daylight deferral. Persisted high-water marks are retained: the earlier duplicate can conservatively suppress the next evening, rather than resetting history. Focused presence/home/app and GameTimeTests checks passed; fullCI is running. Corrected live midnight proof remains unearned until a future real night.

**agent:codex/explore-night** at 2026-11-04T20:09:33Z

Installed f254e95 with deployment preflight/checksums/receipt success; just check vm online restored Tas/private gameplay. FullCI203s and committed review08590de..HEAD clean. Native live status now carries coherent calendarFraction0.18173056,day215 alongside smoothed dayFraction0.272,proving the new native telemetry is installed. presence.bedtime=shout persisted; final exact audit20:05:40Z..20:08Z (.checks/audit-window-1RhaJy.jsonl) is retained. No new real-night/midnight invitation was attempted after this correction, so corrected live rollover proof remains explicitly unearned. The previous duplicate is documented and covered by the exact regression; durable history was retained. Tas is awake,online,at home with injurySeq0; VM lease released.

## Summary

Implemented on PR179, deployed plugin/controller f254e95; full just check ci and committed review passed. Installed f254e95 with deployment preflight/checksums/receipt success; just check vm online restored Tas/private gameplay. FullCI203s and committed review08590de..HEAD clean. Native live status now carries coherent calendarFraction0.18173056,day215 alongside smoothed dayFraction0.272,proving the new native telemetry is installed. presence.bedtime=shout persisted; final exact audit20:05:40Z..20:08Z (.checks/audit-window-1RhaJy.jsonl) is retained. No new real-night/midnight invitation was attempted after this correction, so corrected live rollover proof remains explicitly unearned. The previous duplicate is documented and covered by the exact regression; durable history was retained. Tas is awake,online,at home with injurySeq0; VM lease released.
