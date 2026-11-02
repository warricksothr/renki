---
schema: 4
id: TKT-01M66PMAQSBK2TH076X7M5FGAM
title: Add bounded expedition preparation, survey and return routines
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - plugin
  - movement
  - live
assignees: []
milestone: M7
parent: TKT-01M66PFAYA1NTW2JHNRDKBK42E
origin: null
dependencies:
  - TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
  - TKT-01M62N2557R401S3E6JECSAPN4
  - TKT-01M62VXZ0J9Q2WH4XN4E6N7F27
  - TKT-01M6EFQNT4D9QAMN9E0S7EECRD
  - TKT-01M6EFQNWT6P9JJA3KVHFSQA2F
  - TKT-01M6EFQNYKRHT3AP5CEJNVVSFX
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T15:25:07Z
updated_at: 2026-11-02T15:59:32Z
created_by:
  id: agent:codex/renki-exploration
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
extensions: {}
---

## Description

Implement the supplied land-expedition behaviors proposed in docs/EXPLORATION.md after M2/M3: validate supplies/gear/readiness and anchor, travel one bounded leg, report actual new survey evidence, reassess and return with a reserve. Native C# owns movement and immediate safety. Resolve waypoint versions explicitly and distinguish successful outbound traversal from a proven reversible return. Missing crafting, boats, portals and camp-building capabilities are separate work. This ticket remains draft.

### Split (rollout step 3)
Umbrella for three one-PR children:
- TKT-01M6EFQNT4 (Check expedition readiness and report specific shortfalls): preparation, offline
- TKT-01M6EFQNWT (Native survey task: walk one bounded leg and report it): native leg reusing plugin/Renki.Core/Tasks/Scouting.cs, with one live check
- TKT-01M6EFQNYK (Return to the expedition anchor within a reserved budget): turn-back, arrival proof and one bounded alternative

### Files
- internal/goal/return.go, internal/goal/food_stock.go, internal/home/waypoints.go, plugin/Renki.Core/Tasks/Scouting.cs, plugin/Renki.Core/Protocol/TaskArgs.cs, plugin/Renki.Core/Movement/Arrival.cs

### Verification
`just check controller test-packages ./internal/goal ./internal/protocol`, `just check plugin test-core ScoutingTests`, and the survey child's live `just vm task-check` runs (VM lease). Done when the three children are done.

## Acceptance criteria

- [ ] Prepare against explicit world/area/anchor/supply and native permission constraints, reporting missing prerequisites honestly
- [ ] Travel/survey reports actual scoped coverage and measured travel, handles supported elevation and rejects unsafe unsupported travel modes
- [ ] Return reserve and alternative/blocked return behavior are bounded, cancellable and tested without requiring player or enemy staging
