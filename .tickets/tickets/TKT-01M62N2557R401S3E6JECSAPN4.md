---
schema: 4
id: TKT-01M62N2557R401S3E6JECSAPN4
title: "M3: the bot survives danger and stands with its owner"
type: epic
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
assignees: []
milestone: M3
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-resume
  branch: t3/milestone-evidence
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 552c491e6b0ec2cac344cbb99a4e38a529ff19ef
  session: null
  claimed_at: 2026-10-30T13:45:00Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:45Z
updated_at: 2026-10-30T13:45:00Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

M3 makes the bot safe to take out and useful in a fight. On 2026-10-28 it was killed twice by creatures while standing about and drowned once on a trip, and it never fought back or ran. M2 makes it work; M3 keeps it alive while it does, and lets it help.

The plugin's safety switch for this (Safety.AllowCombat) is off by default and stays that way until an owner turns it on.

## Acceptance criteria

- [ ] Hurt and outmatched, the bot retreats toward its owner or home instead of standing still
- [ ] With combat allowed, it fights what attacks it or its owner, and stops when told
- [ ] It remembers where it was hurt and avoids those places when it can

## Implementation plan

Continue the owner-requested M3 completion. Danger memory/site preference/map layer is implemented and deployed. Native retreat, ground survey, scouting, kiting and gated melee exist. Obtain positive live melee/block/dodge and retreat evidence using bounded automatic supervisors; resolve actual failures without requiring owner staging or clearance. Then implement conservative self/owner defense selection with explicit targeting/damage evidence, species/equipment/health/stamina limits and immediate stop; preserve AllowCombat off by default. Do not infer winnability from hpFrac/stars alone or turn passive wildlife hostility into a fight. Route avoidance beyond target preference and broad milestone acceptance remain open.
