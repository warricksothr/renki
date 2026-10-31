---
schema: 4
id: TKT-01M6A1D13A2A9WXTEZH9MS1WZA
title: Avoid repeating stalled stair approaches on raised-home return
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - movement
  - safety
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-31T22:31:06Z
updated_at: 2026-10-31T22:31:06Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

The western food trial gathered live mushrooms, but its first home return t-12 timed out after120s. The bounded cleanup return t-14 eventually reached verified shelter without injury. Full evidence is .checks/marker-food-20261031T222103Z (especially t-12.json/t-14.json); this is a navigation failure, not an empty resource site.

Near [-48,32.5,7.8], native local search repeatedly found raised entrance routes via approximately [-47.62,32.87,8.13], [-47.62,33.13,8.63], [-47.62,33.47,9.13] toward [-39.77,36.09,11.2]. Actual walking stalled and replanned similar prefixes while keeping feet on support. Inspect capsule/support projection and how failed execution updates RouteRefusals before altering tolerances. Use learned refusals or a newly checked alternative; do not bypass floor, water, collision or injury guards. The successful second return is evidence of recoverability, not proof that the first timeout is fixed.

## Acceptance criteria

- [ ] Replay the captured raised-home approach without repeating a stalled local prefix
- [ ] Preserve dry-ground, support, hostile and floor checks while selecting a checked alternative
- [ ] Earn a bounded live return from the western food site without manual routing
