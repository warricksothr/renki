---
schema: 4
id: TKT-01M6GXWKM2SC3K27C1JE98Z0AV
title: "Travel: allow checked shallow-water wading without swimming"
type: bug
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - movement
  - plugin
  - protocol
  - live
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/home-survey-delivery
  branch: t3/map-home-location
  worktree: /home/sothr/.t3/worktrees/renki/t3-b02dd361
  commit: de0457496e394e5962c124ed0ddb6553948eece1
  session: null
  claimed_at: 2026-11-03T14:44:43Z
  expires_at: null
archive: null
created_at: 2026-11-03T14:44:23Z
updated_at: 2026-11-03T15:08:38Z
created_by:
  id: agent:codex/home-survey-delivery
  name: ""
updated_by:
  id: agent:codex/home-survey-delivery
  name: ""
extensions: {}
---

## Description

During the owner-observed old-house to new-home return on 2026-11-03, native t-66 made supported southbound progress but stopped at [-47.32,30.94,-117.91] with bounded dry bank departure time_limit. Reference water is 30; conservative dry planning requires another clearance buffer and does not model supported shallow wading. Evidence .checks/task-mbs1nked.json and .checks/new-home-bank.json. HP stayed full and injurySeq stayed 16702.

The owner explicitly requested distinguishing safe walkable shallow water from swimming. Add an explicit checked-wading travel mode. Preserve strict dry survey/household contracts. Use native swimmer-depth configuration with a conservative reserve and current local liquid heights; validate actual support, slopes, continuous short steps, body/obstacle/portal clearance and hostile exclusion. Grounded immersion alone is not swimming. Actual swimming, missing support, invalid depth configuration and excessive depth must stop this mode without enabling swimming/shore fallback. Existing direct stop/newer-order behavior applies.

Validate with core boundary and captured-bank regressions, wire argument agreement, real plugin build and full gate/review. Deploy under the lease, then attempt the changed southbound route with actual health, support, immersion and native receipt evidence. New-home arrival and a free-bed claim remain separate live outcomes; do not invent either.

## Acceptance criteria

- [x] Explicit wading mode validates identically in Go and C# and preserves strict dry scoped movement.
- [x] Planning and live execution admit supported shallow steps using current liquid height and native swimming threshold reserves.
- [x] Missing support, discontinuous footing, excessive depth, swimming, unsafe slopes, obstacles, portals and hostile corridors remain refused.
- [x] Focused regressions, the full gate and committed review pass.
- [ ] Changed native southbound travel has retained receipts and health/support/water evidence, or an exact remaining refusal is recorded.

## Implementation plan

Add explicit move_to wade:true, requiring supported dry/hostile-avoiding coordinate travel and disallowing scoped familiarization or combat recovery. Advertise a native feature and refuse dispatch to older plugins. Derive a maximum 0.8 m supported wading allowance from the local character swim depth minus the game's 0.4 m swim offset and a further 0.3 m reserve. Account for settling and validate current liquid heights at supported endpoints, midpoints, short prefixes and actual body footing. Wading bypasses strict dry bank/shore recovery only while grounded, not swimming and inside its measured limit; it never dispatches swimming input. Apply the same mode to local/navmesh/reused-route probes and reset it on every goal. Preserve default dry semantics and survey admission. Add captured-bank, depth/wave, support and protocol regressions, then build/gate/commit/review/deploy and supervise one changed native return. Record partial travel and any unavailable bed separately.

## Notes

**agent:codex/home-survey-delivery** at 2026-11-03T14:54:06Z

Implemented in 98481da. Explicit move_to wade:true is feature-gated (supported_wading_v1) and requires dry/hostile-avoiding coordinates, without scoped survey or combat recovery. The mode uses native character swim depth minus the game offset and reserves, at most0.8m; actual planning clearance reserves another0.15m. All support/normal/step, hostile, obstacle and portal checks remain in the shared physical pipeline. Planning, navmesh validation, reuse and immediate execution use the same depth policy. Each physics step checks actual footing/grounded/swimming state before arrival or input. A reduced native allowance refuses rather than retaining an unsafe plan. Strict dry behavior is unchanged.

WadingTests passed14 cases in .checks/20261103T144805Z-8YnGJo.log. Captured bank body/footing establish the original refusal; the following step in that test is explicitly synthetic. Focused Go argument/feature tests passed in .checks/20261103T144805Z-QpLMzF.log, including supported-plugin dispatch and old-plugin refusal. Existing dry/bank/scope tests passed. Complete just check ci passed214s in .checks/20261103T144842Z-BgQg2G.log:60/60 panel scenarios, controller race tests, real plugin DLL and full core/interop/tooling. Committed just check review passed with no findings in .checks/20261103T145224Z-VPCyxU.log. PR #176 contains the change.

Full deployment is loading the new plugin. Live shallow movement, new-home arrival and bed assignment are not earned by these offline checks.

**agent:codex/home-survey-delivery** at 2026-11-03T14:58:20Z

Build98481da deployed successfully, then native t-1 on the new controller run walked about12m from grounded 0.3m-deep water onto dry ground without observed injury. It stopped at[-52.96,30.64,-130.23], supported at30.62, with grounded=true and swimming=false. Evidence .checks/task-vgfo67v1.json and .checks/new-home-wading-1.json. This is partial shallow movement, not new-home arrival.

The aggregate refusal can only have come from the character-contact normal in that recorded state. Installed Character source confirms GetLastGroundNormal returns m_groundContactNormal, not m_lastGroundNormal; UpdateGroundContact clears the former. Replaced that transient input with a fresh footing query using walking=true, preserving the same0.8 normal limit at the actual measured surface. Core Supported accepts only already validated footing; missing/steep ray results remain null. Added a captured-body/footing regression with synthetic flat versus steep ray normals, and precise runtime refusal reasons/support evidence. just check plugin test-core 'WadingTests|FullyQualifiedName~SupportSelectionTests' passed in .checks/20261103T145621Z-A7s4wD.log. Full gate for the correction is running. Do not count the first partial pass as validation of the corrected guard.

**agent:codex/home-survey-delivery** at 2026-11-03T15:08:38Z

Build3eb180f passed full just check ci (166s), committed just check review and checksum-verified full deployment. Native return t-1 travelled roughly350m farther toward home without observed injury, then stopped57m short. Evidence .checks/task-h56xv1tf.json and .checks/new-home-corrected-3.json. The recorded refusal was body[-188.95,29.49,-412.48], footing29.45, water30.100942611694336, grounded true, swimming false. Depth crossed the0.65m planning allowance by less than1mm but remained below the actual0.8m limit.

The follow-up separates actual current depth from future-step settling reserve. Supported current body/footing may use the0.15m reserve; checked future midpoints/endpoints must recover it. The recorded-wave regression also rejects a flat future step and depth beyond0.8m. Focused just check plugin test-core WadingTests passed in .checks/20261103T150631Z-qgBi60.log; full gate is running. Latest observation15:08 shows idle drift to[-189.95,29.12,-412.8], still grounded, full38.8HP, injurySeq0 and a nearly expired meal. Reobserve before retry; arrival and bed are not yet earned.
