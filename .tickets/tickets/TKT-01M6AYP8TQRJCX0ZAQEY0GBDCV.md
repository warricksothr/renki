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
updated_at: 2026-11-01T07:33:54Z
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
- [x] Focused checks and full CI pass; native discovery and navigation results are recorded

## Implementation plan

Finish the in-flight collision-face implementation, retaining existing solid primitives and authoritative support/body checks. Index bounded readable building meshes and tilted boxes by broad-phase envelopes, validate transforms every frame and include mesh changes in route geometry revision. Observe portal network links and optional known target coordinates; deep-copy durable per-world records and regress reconnect, retag, unlink, unknown destination and rollback cases. Reject accidental portal-trigger crossing through both cache and live queries. Document distinct walking and portal transition edges; automatic portal travel awaits inventory, arrival and return-path verification. Run focused checks, full just check ci, commit and just review, deploy and retain native navigation/discovery results.

## Notes

**agent:codex/renki-milestones** at 2026-11-01T07:07:03Z

Implemented bounded actual building collision faces, broad-phase spatial indexing, live transform/lifetime validation and aggregate route geometry revision. Native walking now includes portal triggers in lower capsule casts/overlaps and ignores unrelated triggers; cache is an optimization only. Observed portal network links and optional known target coordinates flow through shared protocol, per-world durable landmarks and map tooltips. Shared wire fixture plus restart, unlink/retag, unknown-target, copy isolation and persistence rollback regressions pass focused checks. docs/PORTALS.md documents native entry-trigger semantics and a separate future verified travel transition; automatic portal traversal is not yet implemented. Native verification awaits the reviewed deployment and home return.

**agent:codex/renki-milestones** at 2026-11-01T07:12:19Z

Committed651d04f passed full just check ci and just check review f303e66..HEAD, then installed with verified package/DLL checksums while keeping Tas offline. A subsequent final code audit found ordinary navmesh/direct body movement can bypass dry planning probes; added an independent native input-layer portal cast/overlap guard with bounded refusal traces. Rerunning the gate and review before native startup; no portal traversal or safe-arrival result claimed.

**agent:codex/renki-milestones** at 2026-11-01T07:21:42Z

Build e19ce26 passed full just check ci and committed reviews, was pushed in PR147, and installed with controller/plugin checksum verification. Native .checks/recovery-20261101T071741Z observed two base portals: darkforest is linked to -192312150:9683 but has unknown target coordinates; sportal is unlinked. Current and offline memory evidence: .checks/portal-native-20261101T0718.json and .checks/portal-offline-20261101T0720.json. Return failed after0.9m without injury, final46.2/46.2HP; Tas was logged out and controller/panel retained. Audit .checks/audit-window-E2ZUAi.jsonl shows checked bank departure completed, navmesh rejected sliding-support candidates, and wave water30.122m later re-entered bank recovery after its original aggregate8s deadline expired during planning. Native doorway completion and deliberate trigger-avoidance checks remain unearned; no traversal or safe arrival claimed. Follow-up TKT-01M6AZ7DJYGTWDR9G9ZP0MA20M — Traverse known portals with verified arrivals — is filed in draft with an explicit dependency on this observation work.

**agent:codex/renki-milestones** at 2026-11-01T07:27:11Z

Offline verification exposed a separate presentation gap: the session-only landmark endpoint and saved atlas returned no remembered objects after controller restart. The saved-atlas API now returns the selected world and bounded viewport landmarks (at most512), plus known contents for mapped chests; the panel uses those records as remembered markers. Focused selected-world/viewport/bound/chest-content checks pass, as do full just check ci, final panel lint/verify and the focused offline-fog UI check. An initial broad app-package check timed out in the existing Marco presence test; its isolated rerun and the subsequent complete race-enabled CI passed. Native offline response verification follows the controller-only deployment, with Tas left offline.

**agent:codex/renki-milestones** at 2026-11-01T07:33:54Z

Saved-atlas follow-up233430e passed committed review and was deployed with Tas offline. The selected DizzysHellheim viewport now returns both remembered portal records and11 mapped chest inventories after controller restart; successful native response is retained in .checks/portal-offline-20261101T0720.json. This supersedes the earlier pending offline-verification note. Native accidental-entry avoidance remains untested because the shore return still blocks reaching the base floor; its acceptance criterion stays open.
