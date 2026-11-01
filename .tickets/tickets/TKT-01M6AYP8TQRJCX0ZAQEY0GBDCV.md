---
schema: 4
id: TKT-01M6AYP8TQRJCX0ZAQEY0GBDCV
title: Cache building collisions and remember portal links
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - movement
  - controller
assignees: []
milestone: M3
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-milestones
  branch: t3/pantry-maintenance
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: f303e664a3ae3e6d5fcac648ab7fd1fdd344b52f
  session: null
  claimed_at: 2026-11-01T07:03:09Z
  expires_at: null
archive: null
created_at: 2026-11-01T07:02:57Z
updated_at: 2026-11-01T07:12:19Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Complete the requested cached building-part collision model and portal observations. Preserve actual mesh openings, ramps, separate floors and live physics authority; invalidate changed geometry. Remember observed portal connections and known destinations per world, expose them in map tooltips, and prevent normal routes from accidentally entering portal triggers. Document controlled portal traversal as a distinct verified transition.

## Acceptance criteria

- [x] Cached actual collision faces preserve openings and floor separation with bounded storage and change invalidation
- [x] Observed portal links survive memory restart and update without inferring destinations from matching tags
- [ ] Ordinary native navigation refuses portal triggers even before cache discovery
- [ ] Focused checks and full CI pass; native discovery and navigation results are recorded

## Implementation plan

Finish the in-flight collision-face implementation, retaining existing solid primitives and authoritative support/body checks. Index bounded readable building meshes and tilted boxes by broad-phase envelopes, validate transforms every frame and include mesh changes in route geometry revision. Observe portal network links and optional known target coordinates; deep-copy durable per-world records and regress reconnect, retag, unlink, unknown destination and rollback cases. Reject accidental portal-trigger crossing through both cache and live queries. Document distinct walking and portal transition edges; automatic portal travel awaits inventory, arrival and return-path verification. Run focused checks, full just check ci, commit and just review, deploy and retain native navigation/discovery results.

## Notes

**agent:codex/renki-milestones** at 2026-11-01T07:07:03Z

Implemented bounded actual building collision faces, broad-phase spatial indexing, live transform/lifetime validation and aggregate route geometry revision. Native walking now includes portal triggers in lower capsule casts/overlaps and ignores unrelated triggers; cache is an optimization only. Observed portal network links and optional known target coordinates flow through shared protocol, per-world durable landmarks and map tooltips. Shared wire fixture plus restart, unlink/retag, unknown-target, copy isolation and persistence rollback regressions pass focused checks. docs/PORTALS.md documents native entry-trigger semantics and a separate future verified travel transition; automatic portal traversal is not yet implemented. Native verification awaits the reviewed deployment and home return.

**agent:codex/renki-milestones** at 2026-11-01T07:12:19Z

Committed651d04f passed full just check ci and just check review f303e66..HEAD, then installed with verified package/DLL checksums while keeping Tas offline. A subsequent final code audit found ordinary navmesh/direct body movement can bypass dry planning probes; added an independent native input-layer portal cast/overlap guard with bounded refusal traces. Rerunning the gate and review before native startup; no portal traversal or safe-arrival result claimed.
