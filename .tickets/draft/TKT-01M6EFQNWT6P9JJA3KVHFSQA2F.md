---
schema: 4
id: TKT-01M6EFQNWT6P9JJA3KVHFSQA2F
title: "Native survey task: walk one bounded leg and report it"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - protocol
  - movement
  - live
assignees: []
milestone: M7
parent: TKT-01M66PMAQSBK2TH076X7M5FGAM
origin: null
dependencies:
  - TKT-01M6EFQN91YGXHVP9NF32BK21W
  - TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
  - TKT-01M62N2557R401S3E6JECSAPN4
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
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

Rollout step 3, travel and survey. Add a native `survey` task that walks one bounded leg toward a controller-chosen vantage point and reports what happened. It reuses plugin/Renki.Core/Tasks/Scouting.cs, which already does checked 16 m legs, a leash and a bounded return, but is reachable today only through the attack task's kite arguments (plugin/Renki.Core/Protocol/TaskArgs.cs refuses `searchRadius` without kite=true). The survey task has no target and never attacks; existing native retreat keeps priority.

The result carries the trail and measured travel from TKT-01M6EFQN91 (Report and keep the ground the bot actually walked), the final position and an end reason.

### Why it matters to M7
Every leg of goal 7.3 is one survey task. Without it the host has no native routine that moves outward without hunting.

### Files
- plugin/Renki.Core/Tasks/Scouting.cs, plugin/Renki.Tests/ScoutingTests.cs, plugin/Renki.Tests/ScoutFootingTests.cs
- plugin/Renki.Core/Protocol/TaskArgs.cs, plugin/Renki.Plugin/KiteTask.cs (where scouting runs today), plugin/Renki.Plugin/BodyTasks.cs
- internal/protocol/task.go and a fixture under internal/protocol/testdata/
- docs/BRIDGE-PROTOCOL.md

### Verification
- `just check plugin test-core ScoutingTests`
- `just check controller test-packages ./internal/protocol`
- Live (takes the VM lease): `just vm deploy`, then 3 runs of `just vm task-check survey '{...}'` from an owner-approved anchor with combat off; save the outputs and a before/after `GET /api/atlas/coverage` under .checks/

## Acceptance criteria

- [ ] `survey` arguments are validated (vantage inside the authorized circle, leg at most 60 m, work at most 90 s, leash at most the circle radius) and invalid ones are refused with a reason; one fixture under internal/protocol/testdata
- [ ] The task never sends an attack; a threat inside retreat range ends the leg with reason `threat` and native retreat takes over (test)
- [ ] The result reports trail, measured travel, final position and one end reason: arrived, blocked, water, unsupported, threat, timeout or stopped
- [ ] Water, unsupported footing and drops beyond the existing Scouting limits end the leg rather than being crossed (tests)
- [ ] Live: 3 task-check runs each end with a valid reason, and in at least 2 the measured-coverage query rises; evidence under .checks/
