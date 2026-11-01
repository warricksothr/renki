---
schema: 4
id: TKT-01M6BPPVDE5S8Q5A61N218JXDM
title: Peel isolated enemies from groups with native pursuit checks
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - safety
  - movement
assignees: []
milestone: null
parent: TKT-01M62N25826TRP761PYVVC7RKY
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-01T14:02:42Z
updated_at: 2026-11-01T14:02:42Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Build one bounded native group-peel episode following docs/WORLD-EVENTS.md: prepare equipment/food/escape, disengage, select a peripheral target, measure pursuit for the entire group, confirm isolation, fight one and reassess. A closer enemy is not necessarily isolated. Owner-authoritative targeting, event provenance and effective player-hunt mode are available in perception; remote/lost members remain uncertain. Groups that stay together need bounded escape/reposition instead of infinite lure or a larger melee crowd limit. Support Greydwarf-specific preparation/combat before testing the captured six-Greydwarf/three-Greyling group. Keep movement and combat in C#, persistent goal orchestration in Go, and diagnosis visible to agent tools and panel.

## Acceptance criteria

- [ ] Preparation checks enemy support, gear, food, stamina, fighting ground and a checked escape.
- [ ] Isolation uses temporal whole-group pursuit evidence and never treats remote or lost targets as disengaged.
- [ ] A second pursuer, fresh spawn, lost observation or blocked escape aborts safely under bounded work and retry budgets.
- [ ] One verified kill is followed by readiness and loot reassessment; checkpoints never replay uncertain attacks.
- [ ] Deterministic and natural live checks progress from one supported enemy to two before larger mixed groups.
