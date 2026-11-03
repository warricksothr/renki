# Handoff

2026-11-03, agent:codex/home-survey-delivery, branch t3/map-home-location.
Delivery: [PR #176](https://github.com/warricksothr/renki/pull/176).
Runtime code deployed: **8f662f6**. Later documentation commits do not require
another game restart. The owner explicitly authorized filing, grooming,
promotion and implementation while away; the earlier planning-only hold ended.
No merge or CI monitoring was requested.

## Later runtime update: fed, then automatic Loot entered a portal

This update supersedes the inside-house/0.5HP snapshot below. The owner provided
food: an auto-picked Raspberry at15:33:47Z was consumed as t-36, restoring health.
Existing passive timer **Loot** then left the house to collect Wood t-37 near
Nimah, followed by Resin t-38. At15:36:24 the final input guard refused
portal_cast near[-187,30.97,-459.99], but the task kept running. It tried Jump at
15:36:27 and Strafe at15:36:30, shifted closer and teleported through darkforest
at15:36:32. Audit `.checks/audit-window-FfbfFj.jsonl`. This was an automatic Loot
run, not a survey or a new explicit owner relocation order.

Loot is now **disabled persistently** through the normal behavior API
(`.checks/new-home-loot-paused.json`) to prevent further automatic excursions.
Do not re-enable it before reviewing this incident and checking current owner
intent. Latest staging is grounded on structure near[414.45,31.65,-1175.96],
BlackForest,940m from unchanged saved home, no observed local hostiles. Raspberry
was the only meal; reobserve its remaining time and health. A direct three-second
normal movement attempt to the portal center stayed put and timed out, with
portal_cast refusal (`.checks/task-o4knlg4c.json`). Do not bypass that guard.
Remote portal1:161078 tag darkforest reports connected target1:161081 at
[-188.09,30.97,-459.47], but controlled traversal/return is not implemented.
The earlier new-home arrival remains earned; Tas is no longer inside that house.

TKT-01M6H1A2XPHCAR4F016AE4F8K0 (End the body task when the final portal guard
refuses movement) owns the correction: a final driver guard refusal stands and
fails the current body task in the same step, rather than letting its later
stuck recovery jump/strafe into the portal. Lifecycle tests cover single receipt,
no later ticks, contained cancellation and future independent orders. Native
build/gate/review/deployment and a short blocked-portal probe follow. It does
not implement controlled portal traversal or authorize a return bypass.

## Earlier home arrival and storage attack

Tas reached the owner-saved new home through normal native movement, and is
alive **inside the house**, not at the old home or the shoreline. Latest retained
observation `.checks/new-home-final-observation.json`: position
[-203.58,32.97,-466.37],1.34m from saved home[-204.39,32.97,-467.43], world
DizzysHellheim, UID827096491, grounded/sheltered on structure support, idle,
**0.5/25HP**, injurySeq7002, no active or carried food. Inventory contains the
equipped rag tunic and one Wood. `.checks/new-home-final-map.json` confirms
house entrance1:14767 **closed**. Keep Tas protected; do not resume an exposed
survey or supply search at this health. The game remains online. Release the VM
lease before handing back; acquire/renew it before any further live work.

Three house beds1:14717,1:14716,1:14968 and storage-upstairs bed941891633:6688
all report owned:true, none mine. **No free bed was observed; no bed was claimed
and spawn was not changed.** The house chest1:15016 at[-202.58,32.97,-460.24]
contains4DeerMeat,7RawMeat and1NeckTail, with no ready meal in its snapshot.
Observed heated cooking stations941891633:5001/4997 are back in the storage
building near[-169,31,-472], where Tas was attacked. Other checked chests held
armor, wood, resin, stone, flint, seeds or were empty; this is not proof that all
base chests lack meals. The owner subsequently confirmed giving food, as recorded in the update above.
A free Tas bed is still unconfirmed. Do not invent its location or clear another bed.

## Delivered local familiarization

Implemented in0152cb0 and controller follow-ups, documented in EXPLORATION.md.
The panel's Places & storage → Base area & familiarization saves per-world named
bases with separate sleeping/storage/return bindings. `Tas, familiarize yourself
with home` (also this area/the area/survey home) starts the selected saved base;
`survey status`, stop/newer body orders and panel controls are implemented.
Radius10–250m, total time2–20min and total travel50–4000m are configurable.
A 50–80m base span is not treated as a measured radius.

Selected new home remains centered on the owner's saved home, radius60m,
600s and800m, return home. Normal waypoint creation and CAS base save added
**new home storage west**, actual supported/accessible approach
[-169.13,30.97,-467.75], radius5m. It is bound as storage; sleeping is still
unassigned. Saved base version2026-11-03T15:27:44.0249485Z is retained in
`.checks/new-home-storage-role-saved.json`. Old owner home storage, other markers
and the saved home were preserved. The one storage approach does not claim the
entire building's chest stock or access has been mapped.

The durable survey pins scope/role versions, uses bounded dry hostile-avoiding
native legs, records actual 3D travel/trails and post-receipt supported arrival,
reserves return budgets and stops on unsafe/changed/uncertain work. It separates
4m measured/new/refreshed/unknown/stale terrain from visited ground and observed
objects. Scoped place refresh protects owner edits, pinned roles, suppression
and other bases. It opens no chest and claims no bed. Partial results retain
gaps; loaded terrain is not complete room/floor/container inspection.

**No survey goal has been admitted or run.** Native coverage/waypoint-refresh,
pinned return and sleeping/storage round-trip criteria remain unearned.
Health/food/threat readiness and actual sleeping access are unresolved. Food and
other household routines retain their older local constraints around return/
home; selecting a large base does not teach portals or migrate every routine.
Temporary here anchors and per-run overrides remain deferred.

## Shallow-water change and actual return

The owner watched a southbound strict-dry bank refusal and requested walkable
shallow water. Commits98481da,3eb180f and8f662f6 add explicit coordinate
`move_to` wade:true with dry:true and avoidHostiles:true, feature-gated on
supported_wading_v1. This is distinct from strict dry household/survey travel.
The native character swim threshold, game offset and reserve bound supported
wading to at most0.8m. Current actual body/footing may use the0.15m settling
reserve; future midpoint/endpoints must recover it. Walking normals, support,
short continuous steps, obstacles, portals and hostile checks remain enforced.
Swimming/missing footing/excess depth refuse without a swimming fallback.

GetLastGroundNormal returns a transient contact accumulator which the game
clears; actual footing now uses a fresh walkable-normal ray. Captured-bank,
normal and wave tests retain synthetic future geometry explicitly as such.
The exact sub-millimetre wave boundary is an offline regression, not a claim
that the same wave was reproduced after deployment.

After normal old-house resupply/equipment/cooking,3eb180f walked roughly350m
closer without injury, then stopped57m short on the wave margin
(`.checks/task-h56xv1tf.json`). Idle shoreline drift later deepened immersion
from0.6m to swimming1.6m; ordinary native shore recovery regained the eastern
bank (`.checks/task-tck__bjm.json`). A grounded refusal does not anchor the body
against subsequent physics drift.

Corrected8f662f6 earned supported shallow progress near[-172.35,29.43,-420.78],
reported0.6m water, grounded,25HP, injurySeq0
(`.checks/task-xqzomqku.json`, `.checks/new-home-east-approach-current.json`).
Other checked staging/low-floor receipts are `.checks/task-vpnco5bq.json`,
`.checks/task-s3yjuwha.json` and `.checks/task-z7wbqshh.json`. A dry-ground actual
ray below normal0.8 was refused (`.checks/task-wv7c53_4.json`). Strict dry bank
recovery separately refused the storage floor's buffer; checked wading crossed
that supported low floor. Do not credit ordinary swimming recovery or final
ordinary home arrival to the checked-wading contract.

Storage entrance interaction required both gate halves and fresh open-state
verification; a use receipt alone was insufficient. Ordinary movement also
repeatedly alternated stuck local planning with a usable navmesh while stationary
(`.checks/task-rz3mrike.json`). Checked home movement later refused hostile
corridors and stair support probes (`.checks/task-e306cp1b.json`). Native ordinary
move_to t-31 finally reached the actual saved home floor
(`.checks/task-3wi2_zkp.json`); t-32 closed the entrance and a fresh scan confirmed
it. After inspecting the house chest, t-34 returned to the interior home point
(`.checks/task-kzu6dojq.json`). These earn one-way arrival, not survey round trips.

## Storage attack: keep the operator's mistakes in the record

The operator left opened storage gates open during a food search. Greydwarf
2600298615:830 entered and dealt11.75 at15:28:09.117Z and12.78 at15:28:11.282Z,
leaving0.5HP. This was ordinary chest work, not a survey or shallow-water leg.
Exact retained window: `.checks/audit-window-OzbhE9.jsonl`.

Automatic flee t-25 began at15:28:09.826Z. A direct task-check chest probe t-27
replaced the running flee at15:28:11.609Z, **after both hits**. Automatic retreat
resumed as t-28 within0.22s and cancelled the probe, then failed dry-bank
distance_limit at15:28:21.345Z near[-184.9,30.97,-469.7] on the bridge. The
probe interruption did not cause those prior hits, but stale operator readiness
must not silently override recovery. No survival acceptance is claimed.

## Tickets and verification

Done/released: TKT-01M6FVN1XGRBWA75WJWFMYV2V9 (Base areas: separate sleeping,
storage and return places), TKT-01M6FV695N00A10NRMECFMSAHA (Waypoints: refresh
bot places around a selected area), TKT-01M6GXWKM2SC3K27C1JE98Z0AV (Travel:
allow checked shallow-water wading without swimming).

TKT-01M6FV6E40SBSMJEHT227XKRK2 (Home survey: familiarize with an area and update
its places) remains review/released with native criteria5/7 unticked. Existing
bed TKT-01M62PRDSBCRCPH5S0TGGCMAEY (Claim a bed near home, or tell the owner
there is none) and flee TKT-01M62N256MPAJ6YJC11XJS09RS (Plugin: flee task, and
retreating when health is low) have evidence notes; their other-agent claims
were preserved.

Future drafts: TKT-01M6GYMV97NYP025K9D4C0FRS7 (Routes: save waypoint chains
with durable execution progress), TKT-01M6GYMVB10AP9WBAW7Y176PQC (Movement:
observe ground paths and prefer useful path segments), and
TKT-01M6H0S6Y5WFCW5P59CCY4HWK2 (Keep live task probes from interrupting
automatic retreat). No generic durable ordered itinerary/cursor/resume exists.
The atlas has terrain/biome/water, not ground-paint path observations. Future
path preference must be configurable/directionally useful and preserve physical,
water and hostile constraints. Do not promote these future drafts implicitly.

For8f662f6: focused WadingTests passed
`.checks/20261103T150631Z-qgBi60.log`; just check fmt passed
`.checks/20261103T150751Z-LBnkRM.log`; full just check ci passed164s in
`.checks/20261103T150803Z-vJctm7.log`; committed just check review passed with
no findings in `.checks/20261103T151057Z-sfEh4J.log`; full just check vm deploy
passed77s with checksum verification in `.checks/20261103T151204Z-FYkdau.log`.
Final documentation/ticket gate and committed review follow this handoff edit.

Before further live work read fresh body/health/food/door/bed ownership and check
for newer owner orders. Do not replace active automatic retreat with a probe.
Acquire the VM lease and leave the game online unless the owner requests otherwise.
Never guess an approach floor or claim another player's bed. Live acceptance
needs fed/recovered Tas, a free bed and verified separated-building approaches.
