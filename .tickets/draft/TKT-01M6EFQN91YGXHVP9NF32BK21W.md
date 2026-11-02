---
schema: 4
id: TKT-01M6EFQN91YGXHVP9NF32BK21W
title: Report and keep the ground the bot actually walked
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
  - protocol
  - movement
assignees: []
milestone: M7
parent: TKT-01M62VXZ0J9Q2WH4XN4E6N7F27
origin: null
dependencies: []
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

Visited ground is the second coverage kind in docs/EXPLORATION.md: where Tas actually travelled, including elevation. Each movement task already keeps a `RouteTrail` of grounded positions (plugin/Renki.Core/Movement/RouteMemory.cs: at most 256 points, discarded rather than truncated on overflow), but plugin/Renki.Plugin/BodyTasks.cs reports only its point count (`points`).

Report the trail in the task result (points with x, y, z at the existing 0.5 m minimum spacing, an overflow flag and the measured path length), and keep visited segments per atlas world key in the controller with task id, source and time, bounded by count and age.

### Why it matters to M7
The explore goal counts actual travel (goal 7.3's travel bound, including failed approaches) from these lengths. The return step uses the reversed trail as its one alternative. The panel draws travelled lines distinct from planned ones.

### Files
- plugin/Renki.Core/Movement/RouteMemory.cs and plugin/Renki.Tests/RouteMemoryTests.cs
- plugin/Renki.Plugin/BodyTasks.cs (where the trail count is reported today)
- plugin/Renki.Core/Protocol/Messages.cs, internal/protocol/task.go, a fixture under internal/protocol/testdata/
- internal/atlas/store.go: a bounded visited-segment table (TKT-01M6EFQN74 (Atlas: first-seen time and a measured-coverage query) also migrates the schema; whichever lands second takes the next version)
- docs/BRIDGE-PROTOCOL.md

### Verification
- `just check plugin test-core RouteMemoryTests`
- `just check controller test-packages ./internal/protocol ./internal/atlas`
- `just check ci`

## Acceptance criteria

- [ ] A move_to result carries the trail points (x, y, z), an overflow flag and the measured path length; one fixture under internal/protocol/testdata decodes in Go and matches the C# encoder
- [ ] An overflowed trail is reported as overflowed with no points, never as a silently truncated trail (test)
- [ ] The controller stores visited segments per world key with task id, source and time; a trail on a raised floor keeps its y and is not reported as ground below it (test)
- [ ] Retention is bounded by count and age, documented in docs/ATLAS.md, and a test shows the oldest segments are evicted first
- [ ] Measured travel for a task equals the sum of its stored segment lengths within 1 % (test)
