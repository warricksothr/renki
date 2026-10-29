---
schema: 4
id: TKT-01M62VXZ1JG9BTWZBZT43AZT59
title: "Deliver to the nearest storage: home, a ship or a waypoint"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: null
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies:
  - TKT-01M62VXYZMVEP74QVV9SRKDWGH
  - TKT-01M62VXZ0J9Q2WH4XN4E6N7F27
  - TKT-01M62VXYWJHB5BC9VGCBM0PVXD
  - TKT-01M62VXYY6F094WXQS3FBQ57RF
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-10-29T03:40:48Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Deliver-when-full goes home at first. Later it picks the best storage it
can reach: the base, the ship the bot came on (a ship has a hold), or a
waypoint with chests close by.

Each candidate is scored on distance, the room it has (from chest memory),
whether the way crosses unexplored land (strongly avoided), and danger
zones. When nothing is good enough, the bot says so rather than taking a
long, risky trip.

## Acceptance criteria

- [ ] Deliver-when-full can choose the ship's hold or a waypoint's chests instead of home
- [ ] A route over unexplored land or through a danger zone is avoided where another will do
- [ ] When no storage is good enough, the bot says so
