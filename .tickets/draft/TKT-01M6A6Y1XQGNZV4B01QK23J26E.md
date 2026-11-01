---
schema: 4
id: TKT-01M6A6Y1XQGNZV4B01QK23J26E
title: Recover a household return blocked by hostile clearance
type: bug
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - movement
  - plugin
  - safety
  - vm
assignees: []
milestone: M3
parent: TKT-01M6864352N8E28YCGHA19AN90
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-01T00:07:46Z
updated_at: 2026-11-01T00:07:46Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

A household food trial reached the west mushroom marker and gathered food, but its recorded return route became obstructed by current hostile clearance. The return refused that segment, rejected the navmesh candidate for hostile_corridor, and exhausted the bounded local search. A single guarded startup return encountered the same geometry and creatures and failed safely. This blocks the positive outdoor gather-and-store and subsequent home repair trials.

### Reproduction evidence

World DizzysHellheim; home [-35.9,37.66,6.53], actual home floor feetY36.09. Tas is logged out near [-23.22,30.88,-30.42], HP72.3/72.3, three active foods at logout. Greyling [11,33.22,-9.27], native clearance40m; Boar [-26.82,38.75,-87.8], clearance24m. Local search exhausted4096 nodes amid water, support, body, log-footprint and hostile refusals. Source trial `.checks/food-20261031T235956Z`, audit `.checks/audit-window-KTZ3ii.jsonl`; guarded recovery `.checks/recovery-20261101T000427Z`, audit `.checks/audit-window-RA5Dxt.jsonl`.

Determine whether a physically safe detour actually exists before changing planning. Investigate the interaction of threat envelopes, shore/support geometry, the local node budget and frontier progress requirements; a required detour may initially move away from home. Preserve dry movement, measured body/floor/log boundaries, hostile avoidance and all recovery/survival guards. Do not repeatedly retry an unchanged failed search, clear creatures through cheats, or request a person to stage the route as the default solution. Any bounded wait/replan should depend on changed observations and expose its reason to goals and diagnostics.

## Acceptance criteria

- [ ] Retained geometry and hostile evidence establish whether a safe route exists with the current clearances.
- [ ] A feasible detour can make bounded progress even when its first leg moves away from home, without weakening physical or survival checks.
- [ ] Infeasible or unchanged blocked routes expose an actionable bounded outcome instead of blind retries or manufactured arrival.
- [ ] Tas returns to the correct home floor without injury before repeating outdoor delivery and repair trials.
