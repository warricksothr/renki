---
schema: 4
id: TKT-01M69WH7CNZABZZEYKXAE64A0N
title: Run bounded food work while idle near typed markers
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - safety
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-milestones
  branch: t3/food-acquisition
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: d19708daf64a40ecd06fe2f4ca7d10ffe9ea598a
  session: null
  claimed_at: 2026-10-31T21:06:37Z
  expires_at: null
archive: null
created_at: 2026-10-31T21:06:00Z
updated_at: 2026-10-31T21:17:01Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Implement the user's requested first marker-idle food slice: bounded wild-food collection, preparation of carried raw food, and surplus storage near typed markers. Use the existing behaviour scheduler and body lease, current same-world waypoint definitions, delayed idle triggers, cooldowns and explicit opt-in. Preserve meals and interrupt on owner commands, unsafe observations, injuries or marker edits. Farming crops, hunting and pressure-driven multi-site production remain separate work.

## Acceptance criteria

- [x] Marker idle triggers require fresh same-world floor-aware dwell, have cooldowns, and invalidate queued work on leave/edit/reconnect or commands
- [x] Berry collection, carried-raw cooking and surplus storage use bounded native routines and verified receipts while retaining personal food
- [x] Panel exposes opt-in marker work and last outcomes; unattended jobs do not spin on empty sites
- [ ] Focused replay checks and full CI pass; deployment and earned live results are recorded

## Implementation plan

Use the existing serialized behaviour engine for a marker-scoped idle trigger with fresh observation dwell, per-run cooldown and finite run budget. Pin marker identity/version for queued and active work; discard on world/session changes, leaving the area or marker edits. Preserve command takeover through existing body leases.

Implement a marker_food Go template that chooses bounded edible pickables at berries markers, native cooking of carried raw ingredients at cooking markers, or matched surplus deposits to currently perceived local chests at storage markers. Reuse collection, storage and cooking receipt verification; keep personal meals and reject threats, injury, water, stale observations and capacity issues. Show opt-in enable/disable and outcomes in the marker editor. No hunting, crop cultivation, automatic fire fueling or cross-site production loop in this first slice.

Add replay tests for scheduling, cancellation, marker changes, empty sites, reserves and receipts. Run focused checks, just check ci, commit and just check review before push. Deploy the controller preserving game state, run bounded supervised live food/marker checks, and record earned evidence plus remaining limits.

## Notes

**agent:codex/renki-milestones** at 2026-10-31T21:17:01Z

First marker food slice implemented on t3/marker-idle-food (stacked above t3/food-acquisition). Typed marker idle trigger uses 30-second dwell, floor-aware current waypoint pin, owner-command leases, five-minute cooldown and 12 completed attempts per controller session. Marker food runner collects up to four wild-food targets, cooks up to three carried raw ingredients with matched native delivery traces, or stores surplus in currently perceived in-reach boxes while keeping two meals per kind. Panel provides opt-in controls and outcomes. New reusable just vm marker-food-check supervises health, keeps full diagnostics in .checks, and yields to owner takeover.

Focused Go checks passed (.checks/20261031T211104Z-pUBmDQ.log); desktop/mobile marker UI checks passed (.checks/20261031T211335Z-isGK1q.log). First full just check ci passed (.checks/20261031T211342Z-CwCm6k.log); final gate rerun follows small review fixes. Tas replenished through existing food goal d56ffffc32aced16a5641519, safe home and online (.checks/food-20261031T210551Z). New runner live collection/cooking/deposit checks still pending; no crop cultivation or cross-marker production loop claimed.
