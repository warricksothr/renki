---
schema: 4
id: TKT-01M6A9EVRB6KY9J8K1G9BV4BZT
title: Resolve a multi-threat coastal return bottleneck
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
created_at: 2026-11-01T00:51:54Z
updated_at: 2026-11-01T00:51:54Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

The explicit work-trip return profile in PR #139 admitted a healthy, fed, dry body at[-61.67,33.37,-85.61], about96m from home[-35.9,37.66,6.53]. The native search found no eligible progress leg after3228 expansions/25626 probes, with899 hostile-corridor refusals plus water, body casts, support and obstacle refusals. It logged out without movement or injury; final67.6/67.6HP. Evidence `.checks/recovery-20261101T004838Z`, audit `.checks/audit-window-BDzhju.jsonl`. This is a bounded-search failure, not proof that no physical route exists.

Seven loaded threat envelopes were sampled: Greyling[-69.67,32.82,-47.75], radius40; Boar[-28.98,38.4,-97.8], radius24; Necks[-70.72,29.82,-107.38],[-65.85,30.69,-111.72],[-50.15,30.63,-133.66],[-107.96,30.37,-118.38], radius16; Boar[-111.52,30.22,-121.19], radius24. Guidance did not produce an active estimate; the earlier route wrapper discarded inactive graph diagnostics. PR #139 retains inactive hint/waypoint diagnostics without granting another search or movement permission.

Use the retained terrain and collider evidence to determine whether a safe route exists under current clearances, especially initially-away travel around overlapping circles near the coast, selection of four envelopes from seven threats, and the48m local range. A route may require additional loaded observations or a different bounded regional planner, but neither unknown terrain nor a geometric graph can grant movement. If a route exists, add a captured regression and bounded native progress; otherwise expose which observation or environmental change is needed. Do not retry unchanged orders, relax hostile/physical checks, move the pinned home, use cheats, or make manual creature clearing the default test procedure. Outdoor food delivery and home repair remain blocked until a verified home return.

## Acceptance criteria

- [ ] Retained threats and terrain establish route feasibility without treating geometric hints as physical clearance.
- [ ] Feasible initially-away detours make bounded checked progress, or an actionable infeasible outcome identifies the missing observation or change.
- [ ] A native supervised return reaches the actual home floor without injury before further production or repair trials.
