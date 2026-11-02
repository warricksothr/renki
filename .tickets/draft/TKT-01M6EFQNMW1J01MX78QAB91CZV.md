---
schema: 4
id: TKT-01M6EFQNMW1J01MX78QAB91CZV
title: Rank exploration frontiers from measured coverage
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - movement
assignees: []
milestone: M7
parent: TKT-01M66PMAPGXNJHB49E7WMGEBYS
origin: null
dependencies:
  - TKT-01M6EFQN74X0EDYJAVBJJK2DY8
  - TKT-01M6EFQN91YGXHVP9NF32BK21W
  - TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
  - TKT-01M62N2557R401S3E6JECSAPN4
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:58:32Z
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

Rollout step 2 in docs/EXPLORATION.md. A new pure package, internal/explore, ranks frontier candidates: boundary points between measured and unmeasured ground inside the authorized circle. Score expected new measured samples at a vantage point against outbound and return cost. Unknown ground has an explicit uncertainty cost, never zero. Exclude water (height below the reported water level), steep grade from the 4 m heights, known danger zones and owner-excluded zones.

Inputs are plain values: atlas tiles for the circle (internal/atlas Tiles and the TKT-01M6EFQN74 (Atlas: first-seen time and a measured-coverage query) coverage query), visited segments from TKT-01M6EFQN91 (Report and keep the ground the bot actually walked), danger zones from internal/memory/danger.go, the anchor, the radius and the exclusions. No I/O. internal/goal/areas.go (fixed sector selection for combat scouting) is the precedent for host-owned area choice.

### Why it matters to M7
Goal 7.2 is measured on this package's replay scenes, and every expedition leg in goal 7.3 starts from its choice.

### Files
- internal/explore/ (new package and testdata/ with recorded scenes as atlas Tiles JSON)
- internal/atlas/store.go (Tile type), internal/memory/danger.go, internal/goal/areas.go

### Verification
- `just check controller test-packages ./internal/explore`
- `just check ci`

## Acceptance criteria

- [ ] For each recorded scene the ranker returns at most 8 candidates inside the circle, each with expected gain in m2, outbound cost, return cost and reasons
- [ ] Unknown ground costs more than measured ground, and a candidate whose return cost cannot be bounded is excluded rather than costed as zero (test)
- [ ] Candidates in water, on grade above the configured limit, inside danger zones or inside excluded zones are excluded, one test each
- [ ] A scene where all ground inside the circle is already measured returns `no safe frontier` with a reason
- [ ] internal/explore/testdata holds at least 8 recorded scenes, each with a test that it returns ranked candidates or `no safe frontier` (goal 7.2)
- [ ] Ranking a 250 m radius scene takes under 50 ms in a Go benchmark or timed test
