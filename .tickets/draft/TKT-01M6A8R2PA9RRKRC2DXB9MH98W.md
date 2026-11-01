---
schema: 4
id: TKT-01M6A8R2PA9RRKRC2DXB9MH98W
title: Slice hostile guidance construction within the frame budget
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - movement
  - plugin
  - tooling
assignees: []
milestone: M3
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-01T00:39:28Z
updated_at: 2026-11-01T00:39:28Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

The guided household return in PR #138 succeeded without injury, but its bounded visibility graph is built synchronously when ordinary search exhausts. In `.checks/audit-window-GWZnrB.jsonl` at2026-11-01T00:36:18Z, a16-point graph with602 terrain hints built in13.65ms; the worst measured local slice was15.01ms, above the2ms target. Subsequent guidance builds were0.07–0.23ms when the straight corridor became eligible. This is a retained performance limitation, not a failed return or permission to relax physical checks.

Investigate slicing graph construction/hint sampling across physics updates, or reusing terrain hints with explicit world/load/terrain invalidation. Preserve the ordinary-first bridge policy, at-most-one guided retry,4096-hint bound, same-task ownership, overall deadlines and fresh physical/hostile movement authority. Include construction in measured frame work; do not hide the spike by only timing later A* expansions. Compare the same captured fixture and a live return before selecting the implementation.

## Acceptance criteria

- [ ] Guidance construction yields within the existing frame target with bounded total work and diagnostic accounting.
- [ ] Ordinary-first bridge handling, initially-away detours and live physical authority remain covered.
