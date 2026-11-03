---
schema: 4
id: TKT-01M6FT1ZVWC237RH45JMCR8Z9J
title: "Follow: go through the portal the leader just used"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - movement
  - live
assignees: []
milestone: null
parent: TKT-01M6864352N8E28YCGHA19AN90
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-03T04:18:10Z
updated_at: 2026-11-03T04:18:10Z
created_by:
  id: agent:claude-code/passive-deploy
  name: ""
updated_by:
  id: agent:claude-code/passive-deploy
  name: ""
extensions: {}
---

## Description

Live, 2026-11-02 (build a892cfa): Nimah went through a portal while Tas was following. Tas stopped, the follow failed with `target_lost` ("Nimah is gone") after FollowTask's 5 s LoseAfter, and the home keeper walked Tas home. Follow knows nothing about portals, and ordinary movement is deliberately refused at portal triggers: the route planner (WorldNavigation, `portal_trigger_*`) and the last movement guard (`WorldNavigation.PortalClear`, called from Driver.cs:64).

This is a narrower case than TKT-01M6AZ7D (general portal traversal): the owner chose the portal, and Tas copies them.

### Proposed approach

1. **Notice the portal.** FollowTask remembers the leader's last-seen position. If the leader vanishes within about 2 m of a connected portal (TeleportWorld with a target), treat it as "went through", not "lost".
2. **Allow only that portal.** Walk into that one portal. A scoped allowance in the guard admits only its collider, for the rest of this follow, and clears on finish or cancel. Every other portal stays refused.
3. **Respect cargo rules.** The game refuses a player carrying non-teleportable items (TeleportWorld.Teleport, `$msg_noteleport`). If Tas stands in the trigger and isn't teleported within a few seconds, fail with a distinct reason and detail ("can't take my cargo through the portal"), not `target_lost`, and don't send Tas home silently.
4. **Wait out the trip.** The lose timer doesn't run while Player.IsTeleporting(). On arrival, drop the local obstacle caches, look for the leader with a fresh and somewhat longer timer, then continue following.
5. **Cover a pickup in progress.** A passive run cut short when the leader leaves restarts the follow after the leader has already gone. Keep each player's last-seen position plugin-side for a short while, so a follow started just after they vanished still finds their portal.

### Conflicts

TKT-01M6AYP8 (Cache building collisions and remember portal links) owns the portal guard, and agent:codex/renki-milestones has claimed it. Coordinate on WorldNavigation.cs and Driver.cs.

## Acceptance criteria

- [ ] A leader vanishing at a connected portal sends Tas through that portal only; other portals stay refused
- [ ] Cargo the game refuses fails the follow with a distinct reason, and Tas says so rather than walking home
- [ ] The lose timer pauses while teleporting; after arrival Tas finds the leader and keeps following
- [ ] A follow restarted just after the leader vanished (a passive run cut short) still takes the portal
- [ ] Live, under the VM lease: Tas follows Nimah through a portal pair and back
