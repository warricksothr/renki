---
schema: 4
id: TKT-01M6EFQNFNBRJ7X5EJM42NMSDT
title: "Atlas: read-only import preview with validation"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M7
parent: TKT-01M66QF9KD620257SS5A0W987P
origin: null
dependencies:
  - TKT-01M6EFQNDFJ2JSR6YYYGZ5P08N
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

Second half of the portable atlas. `just controller atlas-import-preview FILE [--against atlas.sqlite]` validates an export and reports what it would add, without writing. Validate format version, bounds, size limits, schema, SHA-256 and world/epoch compatibility. Compare against a local atlas: new, equal, conflicting and older samples, with provenance kept. Imported ages are the original measurement times; ingestion time is never treated as a fresh local observation, and imported samples never count as measured coverage.

### Files
- cmd/ (the tool from TKT-01M6EFQNDF (Atlas: export a terrain layer with a versioned manifest)), internal/atlas/, controller.just, docs/ATLAS.md

### Verification
- `just check controller test-packages ./internal/atlas`
- `just check ci`

## Acceptance criteria

- [ ] A round-trip test exports a fixture and previews it against an empty atlas: 100 % of samples match (height to 0.1 m, biome, water, observed time, source), negative coordinates and holes preserved
- [ ] A 1-byte change to the samples file, a wrong format version, or an oversized bounds is refused with a reason (tests)
- [ ] Preview against a different world or epoch reports incompatible and lists nothing to add (test)
- [ ] Preview reports new, equal, conflicting and older counts and never changes the local atlas file (test checks its bytes)
