---
schema: 4
id: TKT-01M6GYMVB10AP9WBAW7Y176PQC
title: "Movement: observe ground paths and prefer useful path segments"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - movement
  - controller
  - live
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-03T14:57:37Z
updated_at: 2026-11-03T14:58:20Z
created_by:
  id: agent:codex/home-survey-delivery
  name: ""
updated_by:
  id: agent:codex/home-survey-delivery
  name: ""
extensions: {}
---

## Description

Owner discussion on 2026-11-03: can Tas recognize paths on the ground and prefer them when they lead in the destination direction? Current atlas captures height, biome and water at4m spacing; it does not record terrain paint. Player-made paths cannot currently be identified as such. This does not request that Tas build or alter paths.

Inspect the installed native terrain paint APIs and effective modpack to distinguish actual path/paved/cultivated/unknown surfaces without guessing channel meanings. Capture observations with world scope, position, freshness, provenance and sufficient local resolution for narrow paths; persist a versioned atlas surface layer if appropriate. Add a configurable modest positive travel-cost preference for known paths heading generally toward the destination. Keep costs positive, cap accepted detours and avoid oscillation at surface boundaries. Geometry, actual supported footing, configured water-depth limits, portals and hostile clearance remain hard constraints. Unknown or stale paint is not a path and missing data must retain ordinary travel behavior. Expose observed surfaces/candidate route preference for inspection, then earn native evidence on an existing real path.

## Acceptance criteria

- [ ] Native observations distinguish path surfaces from cultivated and unknown ground with scoped fresh provenance; existing atlases migrate safely if extended.
- [ ] A configurable path preference ranks directionally useful feasible segments without negative costs, excessive detours or oscillation.
- [ ] Path preference cannot override support, slope, depth, portal, obstacle or hostile constraints and missing surface data preserves ordinary routing.
- [ ] Retained native evidence demonstrates recognition and preference on an existing player-made path without editing terrain.
