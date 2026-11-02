---
schema: 4
id: TKT-01M6EFQNHVMKDVXF42PC5QBP7T
title: "Atlas: GeoPackage export validated by GDAL"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - tooling
assignees: []
milestone: null
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
updated_at: 2026-11-02T16:01:44Z
created_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

docs/ATLAS.md prefers OGC GeoPackage for GIS exchange with explicit local coordinates (undefined Cartesian SRS -1 or a named engineering frame), and requires numeric elevation through the gridded-coverage representation rather than a coloured raster. Add a GeoPackage output to the TKT-01M6EFQNDF (Atlas: export a terrain layer with a versioned manifest) exporter and validate it with GDAL (`gdalinfo`/`ogrinfo`) in a just recipe that skips with a clear message when GDAL is not installed.

Whether M7 needs this, or the manifest format from TKT-01M6EFQNDF is enough for M7, is an owner decision.

### Files
- cmd/ (exporter from TKT-01M6EFQNDF), internal/atlas/, controller.just, docs/ATLAS.md

### Verification
- `just check controller test-packages ./internal/atlas`
- the new GDAL validation recipe through `just check`

## Acceptance criteria

- [ ] The GeoPackage passes GDAL validation and declares no Earth CRS
- [ ] Heights read back through GDAL match the export to 0.1 m, nodata where the atlas has holes
- [ ] The GDAL recipe skips with a message, not a failure, when GDAL is absent, and is documented in docs/TESTING.md
