---
schema: 4
id: TKT-01M6EFQNDFJ2JSR6YYYGZ5P08N
title: "Atlas: export a terrain layer with a versioned manifest"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - docs
assignees: []
milestone: M7
parent: TKT-01M66QF9KD620257SS5A0W987P
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

First half of the portable atlas (docs/ATLAS.md, "Portable formats and broader sharing"). Export the terrain/biome samples for one world key and bounds (at most 512 m per side, as `Tiles` already enforces) from one consistent read transaction. Output a versioned manifest plus a samples file. The manifest holds format version, namespace, world UID as a decimal string, epoch, coordinate frame (local Cartesian metres, +x east, +z north, +y up, 4 m spacing, signed grid keys with floor division), bounds, as-of time, source manifests, row count and the SHA-256 of the samples file. Each sample row keeps grid x/z, height, biome, water, observed time and source. Holes stay absent; no Earth CRS is claimed.

Players, waypoints, containers and server connection details are never included. No upload or publishing.

Expose it two ways: an authenticated read-only `GET /api/atlas/export` (so the VM's atlas can be fetched with `just vm panel GET`), and `just controller atlas-export` over a local atlas.sqlite through a small cmd/ tool beside cmd/audit.

### Files
- internal/atlas/store.go, internal/app/atlas.go, cmd/ (new tool), controller.just, docs/ATLAS.md

### Verification
- `just check controller test-packages ./internal/atlas ./internal/app`
- `just check ci`

## Acceptance criteria

- [ ] Export of a fixture world with negative coordinates and holes produces a manifest and samples file whose row count and SHA-256 match (test)
- [ ] The export contains only terrain/biome samples and manifest metadata; a test with players, waypoints and containers in the stores finds none of them in the output
- [ ] `GET /api/atlas/export` requires authentication and refuses bounds over 512 m per side
- [ ] `just controller atlas-export` writes the same bytes as the endpoint for the same atlas and bounds (test)
- [ ] docs/ATLAS.md documents the manifest fields and format version
