---
schema: 4
id: TKT-01M6EFQNBJCQP8BGKX658XVNBS
title: Report minimap explored cells as map knowledge
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
  - protocol
  - live
assignees: []
milestone: M7
parent: TKT-01M62VXZ0J9Q2WH4XN4E6N7F27
origin: null
dependencies:
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

This carries the original scope of TKT-01M62VXZ0J9Q2WH4XN4E6N7F27 (Know which parts of the map the bot has explored). The minimap knows which areas are explored. The plugin sends that, coarse and only when it grows, so the controller can prefer routes over known ground and avoid sending the bot across unexplored land to reach a goal.

Map knowledge stays apart from measured coverage (TKT-01M6EFQN74 (Atlas: first-seen time and a measured-coverage query)) and visited ground (TKT-01M6EFQN91 (Report and keep the ground the bot actually walked)). A shared map or a cartography table can reveal ground Tas never surveyed, so this layer keeps provenance and never counts as discovery.

### Files
- plugin/Renki.Plugin/TerrainReporter.cs as the model for a new bounded reporter (send only on growth, at most once a second)
- plugin/Renki.Core/Protocol/Terrain.cs and plugin/Renki.Core/Protocol/Messages.cs for the message shape
- internal/protocol/terrain.go, internal/session/terrain.go, a fixture under internal/protocol/testdata/
- internal/atlas/store.go for per-world storage
- docs/BRIDGE-PROTOCOL.md
Read how the installed game's minimap stores explored state before choosing the cell size; do not guess it.

### Verification
- `just check plugin test-core` with a new test class for change detection and bounds
- `just check controller test-packages ./internal/protocol ./internal/session ./internal/atlas`
- Live (takes the VM lease): `just vm deploy`, then `just vm panel GET` on the new read endpoint with Tas in world; save the response under .checks/

## Acceptance criteria

- [ ] The plugin reports explored map cells when they change, with a protocol fixture
- [ ] The controller keeps the explored area per world and can say whether a straight route crosses unexplored land
- [ ] Map-knowledge cells keep provenance (own minimap or shared) and are never counted by the measured-coverage query (test)
- [ ] One live capture after deploy shows explored cells for the current world through the panel API, saved under .checks/
