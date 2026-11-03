---
schema: 4
id: TKT-01M6GXWKM2SC3K27C1JE98Z0AV
title: "Travel: allow checked shallow-water wading without swimming"
type: bug
status: done
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
claim: null
archive: null
created_at: 2026-11-03T14:44:23Z
updated_at: 2026-11-03T15:34:57Z
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
- [x] Changed native southbound travel has retained receipts and health/support/water evidence, or an exact remaining refusal is recorded.

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

**agent:codex/home-survey-delivery** at 2026-11-03T15:34:56Z

Final code is8f662f6, deployed with checksums and native join in .checks/20261103T151204Z-FYkdau.log (just check vm deploy,77s). just check ci passed164s in .checks/20261103T150803Z-vJctm7.log; committed just check review passed with no findings in .checks/20261103T151057Z-sfEh4J.log. The current-vs-future settling correction has the recorded-wave core regression; do not claim that exact sub-millimetre wave boundary was reproduced natively.

Native corrected-mode evidence: t-2 reached the eastern stage (.checks/task-vpnco5bq.json); t-3 made grounded shallow progress to[-172.35,29.43,-420.78], reported0.6m depth,25HP and injurySeq0, then refused the deeper next segment (.checks/task-xqzomqku.json, .checks/new-home-east-approach-current.json). The next checked leg made another supported progress segment (.checks/task-s3yjuwha.json). t-5 stopped on an actual footing ray normal below0.8 on dry ground rather than bypass the slope rule (.checks/task-wv7c53_4.json). t-14 crossed the storage building's low supported floor successfully (.checks/task-z7wbqshh.json). Strict dry bank recovery separately refused that floor's planning reserve; its contract is preserved. These are actual supported shallow/low-floor outcomes and exact remaining refusals, not swimming authorization.

Earlier long-leg3eb180f covered roughly350m farther toward home without injury, then stopped57m short at the wave margin. While idle, body drift deepened from0.6m water to swimming1.6m. Normal native shore recovery, not checked wading, regained dry ground (.checks/task-tck__bjm.json). Grounded wading refusal is not a guarantee the physics body stays put indefinitely.

A later ordinary food search inside storage was attacked by a Greydwarf after the operator left entrance gates open. Damage11.75 and12.78 occurred at15:28:09.117Z and15:28:11.282Z; this is separate from the completed shallow travel. Automatic flee t-25 was briefly replaced by the direct chest probe t-27 after both hits, then resumed as t-28 within0.22s; t-28 stopped on dry-bank distance_limit at the bridge. Audit .checks/audit-window-OzbhE9.jsonl. No survive/combat/survey criterion is earned by that incident.

Ordinary native move_to t-31 subsequently reached the actual saved home floor (.checks/task-3wi2_zkp.json); final inside-house return t-34 succeeded (.checks/task-kzu6dojq.json). The checked home attempt had refused hostile corridors and stair support probes (.checks/task-e306cp1b.json), so arrival is not credited to checked wading alone. Latest observation .checks/new-home-final-observation.json is runtime8f662f6,[-203.58,32.97,-466.37],1.34m from saved home, grounded/sheltered,0.5/25HP, injurySeq7002, no food, idle. .checks/new-home-final-map.json confirms entrance1:14767 closed. Three house beds and one storage bed are owned, none mine/free; no bed was claimed. House chest1:15016 has4DeerMeat,7RawMeat,1NeckTail and no ready meal in its snapshot. Cooking stations remain at the exposed storage building. Leave Tas protected; do not force another exposed search at this health.

Saved the actual supported/accessible storage approach[-169.13,30.97,-467.75] as new home storage west (5m radius), and CAS-bound it to new home storage. Return, home, old owner markers and unassigned sleeping are preserved. Saved base version15:27:44.0249485Z is in .checks/new-home-storage-role-saved.json. Full local survey still has no admitted run: food/health/threat readiness and a free Tas bed remain pending; native coverage/return and separated-building round trips are unearned.

## Summary

Delivered explicit supported shallow-water travel in98481da/3eb180f/8f662f6, with full gate, committed review, checksum-verified deployment and retained native shallow/low-floor movement plus exact refusals. Current-vs-future depth reserve has recorded-wave regression. Ordinary movement separately reached new home; later storage combat left0.5HP. Bed and full survey remain unearned and separately tracked.
