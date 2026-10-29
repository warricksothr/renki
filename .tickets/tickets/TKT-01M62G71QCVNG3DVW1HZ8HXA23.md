---
schema: 4
id: TKT-01M62G71QCVNG3DVW1HZ8HXA23
title: Bot swims into deep water on a straight-line path and drowns
type: bug
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - movement
  - safety
  - plugin
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/m2
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 5613bd8d59db140f6a554db688622cd109609466
  session: null
  claimed_at: 2026-10-29T13:55:41Z
  expires_at: null
archive: null
created_at: 2026-10-29T00:16:03Z
updated_at: 2026-10-29T13:55:41Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The bot walks into deep water and drowns. It happened three times on 2026-10-28 in DizzysHellheim, all in the lake between home (-35.9, 37.66, 6.53) and the ground north of it around z 120–145:

- **18:39**: on "come to me" from 184 m, at hp 0.2 after a fight. Drowned.
- **19:05**: on "come to me" from 325 m, at 25/25. Swam about 30 m and lost 14 hp before it reached the far shore.
- **19:14**: going home after stuck recovery gave up, at 11/25. Drowned 10 s after entering the water.

In every case the navmesh had no path ("no navmesh path; straight line"), so the walker took the straight line, and that line crosses the lake. The bot was running, so its stamina was already low when it started swimming. It never eats, so health it loses stays lost (another way to die, separate from this one).

### What to do

- The straight-line fallback should sample the line for water (the water level against Heightmap.GetHeight). When the line crosses deep water, it should not take it: fail with a reason the controller can report ("there's water in the way"), or route around it once the sparse grid spike (TKT-01M62ACH) lands.
- If the bot is swimming anyway, it should stop running, and it should turn back to the nearest shore when stamina gets low instead of pressing on.
- The controller should not send the bot on a trip home after a give-up when the only route is the straight line that just failed. Today the stuck give-up and the trip home both use the same straight line.

## Acceptance criteria

- [ ] The straight-line fallback does not cross deep water
- [ ] A swimming bot walks rather than runs, and turns for shore when stamina is low
- [ ] A stuck give-up does not start a trip home along the same blocked straight line
