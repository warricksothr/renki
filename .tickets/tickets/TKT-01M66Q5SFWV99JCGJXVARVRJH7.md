---
schema: 4
id: TKT-01M66Q5SFWV99JCGJXVARVRJH7
title: Durable versioned world atlas and portable mapping data
type: epic
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - plugin
  - protocol
  - docs
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: children
references: []
moved_to: null
claim:
  actor: agent:codex/renki-atlas
  branch: t3/atlas-deployment
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 1561eb53c7644a30fed28e7f54df81ed1e16fe45
  session: null
  claimed_at: 2026-10-30T16:08:49Z
  expires_at: null
archive: null
created_at: 2026-10-30T15:34:39Z
updated_at: 2026-10-30T16:08:49Z
created_by:
  id: agent:codex/renki-atlas
  name: ""
updated_by:
  id: agent:codex/renki-atlas
  name: ""
extensions: {}
---

## Description

Design durable mapping that survives controller/plugin runs, merges newer terrain observations into an accumulated atlas, keeps changed versions for a bounded period, and supports portable mapping exchange later. Current terrain is a 256-tile session cache; other memories persist separately. See docs/ATLAS.md for evidence, storage decisions and rollout.

Proposed canonical store: a separate local SQLite atlas with fixed Cartesian sample/tile keys, current materialized state and a bounded change journal. Track stable world identity/epoch, source/provenance, sample freshness, resolution and content hashes. New unknown samples never erase known measurements; unchanged fresh observations renew confirmation without creating history. Separate terrain/biomes, objects/resources, transient conditions, coverage and route evidence with different retention policies.

Sharing remains a future explicit export/import capability: prefer GeoPackage/local coordinate metadata for GIS exchange, numeric raster snapshots where useful, and read-only tile archives for publication. Do not present local game coordinates as WGS84 GeoJSON. No publishing or runtime storage change is part of this design ticket.

## Acceptance criteria

- [ ] Mapping survives restarts under stable world/epoch identity and updates current state without losing unsampled data
- [ ] Changed observations retain bounded history; freshness, logical expiry, quotas and physical storage maintenance are explicit and tested
- [ ] Panel distinguishes current, remembered/stale, unknown and imported evidence without claiming exploration or navigability
- [ ] Portable mapping representation defines local coordinates, provenance, compatibility and selected shareable layers; exchange remains explicit

## Implementation plan

PR #79 merged as 1561eb5. Initial installation found Valheim stopped but the panel controller still held controller.exe open, so SCP could not replace it. Add reusable just vm panel-stop with bounded process wait, and make the full installer call it even when Valheim is already offline. Then land the tooling fix, install, collect a live measured atlas and verify it survives controller reconnect and offline panel viewing. Retain full API evidence in .checks with bounded summaries.

## Notes

**agent:codex/renki-atlas** at 2026-10-30T16:04:17Z

The first three children are implemented: identity/freshness, durable terrain storage and saved-map viewing. The explicit portable export/import child remains draft; no shared publication or imported evidence is enabled. docs/ATLAS.md now distinguishes implemented runtime from the broader format/layer proposal. Local CI, interop, Windows builds and browser fixture checks passed; live deployment proof follows.
