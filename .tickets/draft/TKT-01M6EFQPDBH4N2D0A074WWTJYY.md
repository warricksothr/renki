---
schema: 4
id: TKT-01M6EFQPDBH4N2D0A074WWTJYY
title: "Panel: draw expedition scope, coverage, trail and frontiers"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M7
parent: TKT-01M66PMAV9P2Z8QMR9N90T0TH1
origin: null
dependencies:
  - TKT-01M6EFQP437YT124TQ25299BJV
  - TKT-01M6EFQN91YGXHVP9NF32BK21W
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:58:33Z
updated_at: 2026-11-02T15:58:44Z
created_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
extensions: {}
---

## Description

Rollout step 5, display. On the existing map, show the active expedition: authorized circle and exclusions, pinned anchor, measured coverage gained since start, visited trail (TKT-01M6EFQN91 (Report and keep the ground the bot actually walked)), the current frontier and leg, failed approaches, and remaining time, travel and return reserve. Planned lines, travelled segments and uncertain routes look different. Read-only; reuse the goal summary from `GET /api/goals` and the atlas endpoints.

### Files
- panel/src/controller.jsx (map), internal/panel/static/terrain.js and internal/panel/terrain_test.cjs
- internal/app/goals.go (summary fields), scripts/map-preview.py (fixture preview)

### Verification
- `just check controller test-map`
- `just check panel ci`
- `just controller preview-map --serve` with a recorded expedition fixture
- One `just vm map-snapshot` during a TKT-01M6EFQPB8 (Earn three consecutive bounded expedition passes on the VM) run, saved under .checks/ (read-only)

## Acceptance criteria

- [ ] Each layer above renders from a recorded fixture, and a test shows planned, travelled and uncertain lines use distinct styles
- [ ] Measured coverage gained since start is shown in m2 and matches the coverage endpoint for the same fixture
- [ ] With no active expedition, the map shows none of these layers (test)
- [ ] A map snapshot taken during a live expedition run is saved under .checks/ and linked in a note
