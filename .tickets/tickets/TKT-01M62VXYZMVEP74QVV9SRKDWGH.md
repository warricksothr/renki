---
schema: 4
id: TKT-01M62VXYZMVEP74QVV9SRKDWGH
title: "Waypoints: named places an owner marks"
type: task
status: in-progress
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
claim:
  actor: agent:codex/renki-resume
  branch: t3/named-waypoints
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 2f53d707642e1136d122bfbe3a24efe976af7d18
  session: null
  claimed_at: 2026-10-30T00:20:24Z
  expires_at: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-10-30T00:30:59Z
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

- [ ] Owners can mark, rename and forget named places from chat, per world
- [ ] Waypoints survive a restart and show on the panel map
- [ ] move_to and behaviour parameters accept a waypoint by name

## Implementation plan

User explicitly requested waypoint creation/change/removal and expanded this to web-managed zones and bot-created standard places. Implement persistent per-world named points/circles in the existing shared memory with transactional saves; owner edits/renames/removals take precedence over automatic observed storage/cooking/harvest categories, with deletion suppression. Add owner chat mark/rename/forget/go commands, resolve named move_to destinations centrally before bridge dispatch (including behaviour parameters), show map markers/circles and a map editor for add/move/rename/resize/delete/go, and provide bounded nearby current/remembered observations. Auto-discover only from valid scoped observations, never invent unseen coordinates. Memory dependency is only partly complete, but its durable landmark/container slice needed here is landed; missing/danger inference is not needed and no omission proves depletion. Verify persistence/world isolation, rollback, owner permissions, stale world/order resolution, auto/user precedence and browser edits; run just check ci and committed just check review before landing/deploy. Tas stays offline while developing; harvesting delivery and PvP proximity live checks remain open.

## Notes

**agent:codex/renki-resume** at 2026-10-30T00:30:59Z

Implemented persistent world-scoped named points/circular zones in shared memory, owner chat mark/move/rename/forget/go/list, central named move_to resolution for chat/panel/behaviour parameters, and authenticated web editing with stale-edit protection. Automatic observed storage/cooking/wood/stone/flint/berry/ore roles fill only missing roles; owner edits win and deletion suppresses recreation. Nearby context merges current perception, landmarks and remembered containers with dated stock and world isolation; newer positions override older ones. No zone starts a harvesting job.

Focused persistence, rollback, world separation, owner permissions, auto/owner precedence, stale edits, named wire resolution and world-change tests pass (latest .checks/20261030T003042Z-eAp1hh.log). Native browser fixture checks pass: rename/resize preserves floor Y, refresh preserves unsaved input, map creation with terrain height, dated remembered chest contents, saved-place Go, deletion, and discarding late save after a world change; no runtime errors. Full gate is running again after the container-only context regression fix and packaging docs update. Live persistence/movement/discovery checks remain pending; Tas remains offline.
