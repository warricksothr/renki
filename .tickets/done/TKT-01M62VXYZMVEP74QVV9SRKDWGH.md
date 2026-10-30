---
schema: 4
id: TKT-01M62VXYZMVEP74QVV9SRKDWGH
title: "Waypoints: named places an owner marks"
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
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies:
  - TKT-01M62EP0PDAB02DNJAJ6WS3MYD
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-10-30T00:36:45Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

Owners name places from chat: `<name> mark this as the mine`, `forget the
mine`. A waypoint is a position and a name, per world, kept in memory and
drawn on the panel map. Behaviours and chat can use the name: "go to the
mine", or a `stow` whose chests are at the mine.

The home stays its own thing. A waypoint is any other place worth a name.

## Acceptance criteria

- [x] Owners can mark, rename and forget named places from chat, per world
- [x] Waypoints survive a restart and show on the panel map
- [x] move_to and behaviour parameters accept a waypoint by name

## Implementation plan

User explicitly requested waypoint creation/change/removal and expanded this to web-managed zones and bot-created standard places. Implement persistent per-world named points/circles in the existing shared memory with transactional saves; owner edits/renames/removals take precedence over automatic observed storage/cooking/harvest categories, with deletion suppression. Add owner chat mark/rename/forget/go commands, resolve named move_to destinations centrally before bridge dispatch (including behaviour parameters), show map markers/circles and a map editor for add/move/rename/resize/delete/go, and provide bounded nearby current/remembered observations. Auto-discover only from valid scoped observations, never invent unseen coordinates. Memory dependency is only partly complete, but its durable landmark/container slice needed here is landed; missing/danger inference is not needed and no omission proves depletion. Verify persistence/world isolation, rollback, owner permissions, stale world/order resolution, auto/user precedence and browser edits; run just check ci and committed just check review before landing/deploy. Tas stays offline while developing; harvesting delivery and PvP proximity live checks remain open.

## Notes

**agent:codex/renki-resume** at 2026-10-30T00:30:59Z

Implemented persistent world-scoped named points/circular zones in shared memory, owner chat mark/move/rename/forget/go/list, central named move_to resolution for chat/panel/behaviour parameters, and authenticated web editing with stale-edit protection. Automatic observed storage/cooking/wood/stone/flint/berry/ore roles fill only missing roles; owner edits win and deletion suppresses recreation. Nearby context merges current perception, landmarks and remembered containers with dated stock and world isolation; newer positions override older ones. No zone starts a harvesting job.

Focused persistence, rollback, world separation, owner permissions, auto/owner precedence, stale edits, named wire resolution and world-change tests pass (latest .checks/20261030T003042Z-eAp1hh.log). Native browser fixture checks pass: rename/resize preserves floor Y, refresh preserves unsaved input, map creation with terrain height, dated remembered chest contents, saved-place Go, deletion, and discarding late save after a world change; no runtime errors. Full gate is running again after the container-only context regression fix and packaging docs update. Live persistence/movement/discovery checks remain pending; Tas remains offline.

**agent:codex/renki-resume** at 2026-10-30T00:36:45Z

Landed PR #40 (a1a1a41) after just check ci passed (.checks/20261030T003058Z-xALZEg.log) and committed just check review reported no findings (.checks/20261030T003126Z-N1lZpl.log). just check vm deploy passed with package hashes verified (.checks/20261030T003214Z-5zcY0o.log).

Live DizzysHellheim checks passed: observed automatic storage/cooking/flint/stone; cooking station is visible; create/rename/move/resize on a raised indoor floor; controller restart preserves owner edits; deletion of automatic storage stays suppressed after restart; shared remembered nearby chest stock; saved map publication; named move_to reaches the floor; saved-place Go completes. Zone edits started no harvesting. Evidence .checks/waypoint-live-evidence.json, .checks/task-oumbygrq.json and t-5 in .checks/audit-window-jrBRpq.jsonl. Temporary indoor approach test zone was deleted. Restored the edited storage zone as home storage at [-40.1,36.09,6.8], radius8, centered on an accessible indoor stance rather than the chest collider. Other observed places remain automatic.

Same named home storage destination then passed the outdoor stump return in t-7 (.checks/task-68smxtw6.json). Bot PvP false is exposed live; no nearby owner was present, so protected-bystander harvesting remains unverified. Wood/berry/ore automatic selection has deterministic coverage but no qualifying live sightings in this check. Future jobs can use these anchors; this change does not add zone-scoped harvesting automation.

## Summary

PR #40 landed shared named points/zones, owner chat commands, automatic observed standard purposes, web editing/map markers/nearby memory and named movement. Offline, native-browser and live restart/movement/discovery checks pass; see the latest note and docs/WAYPOINTS.md. Deployed a1a1a41. The live home storage zone also served a successful raised-home stump harvest/collection/delivery/return check. Zone-scoped harvesting jobs remain future work.
