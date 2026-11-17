---
schema: 4
id: TKT-01M7NEG203A7HHNMQE0F8PW3AM
title: "Route memory: plan building routes over ground Tas has walked"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - movement
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
created_at: 2026-11-17T19:07:17Z
updated_at: 2026-11-17T19:07:17Z
created_by:
  id: agent:claude-code/chests
  name: ""
updated_by:
  id: agent:claude-code/chests
  name: ""
extensions: {}
---

## Description

On 2026-11-17 at the new home, the cautious bounded walker gave up three separate ways where an ordinary `move_to` got through:

- **Stairs.** Two home surveys failed their first leg from the upper floor with no travel. Ordinary moves through the stair top and stair foot recorded on 2026-11-04 reached the warehouse.
- **Round a chest row.** A barrel behind a double row failed four bounded approaches to a verified standing spot 5 m away. An ordinary move to that spot went round the row and the barrel opened.
- **Between buildings.** From the east building, six containers 12 to 15 m away in the warehouse failed with "no safe local route" and no standing spot measured.

The base has several buildings, stairs, inclines and upper levels. Each behaviour now works round this on its own (the census walks to a standing spot or toward the container). That does not scale to stowing, sorting, surveys and returns.

### Proposal: route memory in the controller

Keep what the body has actually walked, and plan over it.

- **Record proven links.** Native trails already come back in survey and inspection receipts, and state frames give the body's position. Turn completed walks into per-world links between points, each with its height, so floors stay distinct.
- **Learn connectors.** A link that changes height (stairs, a ramp) or passes a door is a connector. Following an owner through a building teaches the route without anyone marking it.
- **Plan in legs.** For a target on another floor or in another building, plan over proven links and send ordinary moves leg by leg, checking arrival on the intended floor after each. A link that fails is marked and avoided until walked again.
- **Areas.** Group containers and stations by building and floor from the same graph. The storage plan needs these groups anyway.
- **Safety stays native.** Threat, water, fall and ward checks remain in the plugin for every leg.

### Not in scope

Changing the bounded walker's search, or centring on stairs and doorways: TKT-01M6M0HDCE27H971Q6SN08FHQN (Prefer clear building interiors and centered stair approaches) covers the native side.

### To decide first

- Where links are stored (memory.json beside landmarks, or the atlas) and how they age.
- Whether an ordinary multi-leg move is acceptable for unattended runs, or only when an owner started the run.

## Acceptance criteria

- [ ] Completed walks become per-world links with height, persisted across restarts
- [ ] A target on another floor or in another building is reached by planned legs over proven links
- [ ] A failed link is avoided until it is walked again
- [ ] Containers and stations are grouped by building and floor
- [ ] Live: home to warehouse and back, and a census across both buildings, without hand-fed waypoints
