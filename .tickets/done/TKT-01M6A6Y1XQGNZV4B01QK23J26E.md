---
schema: 4
id: TKT-01M6A6Y1XQGNZV4B01QK23J26E
title: Recover a household return blocked by hostile clearance
type: bug
status: done
status_reason: null
priority: high
due_on: null
labels:
  - movement
  - plugin
  - safety
  - vm
assignees: []
milestone: M3
parent: TKT-01M6864352N8E28YCGHA19AN90
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-01T00:07:46Z
updated_at: 2026-11-01T00:38:51Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

A household food trial reached the west mushroom marker and gathered food, but its recorded return route became obstructed by current hostile clearance. The return refused that segment, rejected the navmesh candidate for hostile_corridor, and exhausted the bounded local search. A single guarded startup return encountered the same geometry and creatures and failed safely. This blocks the positive outdoor gather-and-store and subsequent home repair trials.

### Reproduction evidence

World DizzysHellheim; home [-35.9,37.66,6.53], actual home floor feetY36.09. Tas is logged out near [-23.22,30.88,-30.42], HP72.3/72.3, three active foods at logout. Greyling [11,33.22,-9.27], native clearance40m; Boar [-26.82,38.75,-87.8], clearance24m. Local search exhausted4096 nodes amid water, support, body, log-footprint and hostile refusals. Source trial `.checks/food-20261031T235956Z`, audit `.checks/audit-window-KTZ3ii.jsonl`; guarded recovery `.checks/recovery-20261101T000427Z`, audit `.checks/audit-window-RA5Dxt.jsonl`.

Determine whether a physically safe detour actually exists before changing planning. Investigate the interaction of threat envelopes, shore/support geometry, the local node budget and frontier progress requirements; a required detour may initially move away from home. Preserve dry movement, measured body/floor/log boundaries, hostile avoidance and all recovery/survival guards. Do not repeatedly retry an unchanged failed search, clear creatures through cheats, or request a person to stage the route as the default solution. Any bounded wait/replan should depend on changed observations and expose its reason to goals and diagnostics.

## Acceptance criteria

- [x] Retained geometry and hostile evidence establish whether a safe route exists with the current clearances.
- [x] A feasible detour can make bounded progress even when its first leg moves away from home, without weakening physical or survival checks.
- [x] Infeasible or unchanged blocked routes expose an actionable bounded outcome instead of blind retries or manufactured arrival.
- [x] Tas returns to the correct home floor without injury before repeating outdoor delivery and repair trials.

## Implementation plan

The retained terrain overlay shows the short return corridor pinched between the coastline and a40m Greyling clearance. The current Euclidean A* score and mandatory8m reduction in straight-line goal distance can spend the full4096-node bound on an unusable short side, while a safe detour may initially lead away from home.

Add a bounded Core visibility-graph guidance estimate around at most four current threat envelopes (eight conservative ring vertices each). Exclude ring hints on known submerged terrain; unknown hints grant no movement authority. Use this estimate for A* ordering and progress-leg selection, including long detours to nearby goals. Every actual step still passes the existing physics, water, floor, obstacle, refusal and live hostile probes; the actual task arrival remains at its original target. No arbitrary reposition orders, attack permission or survival-clearance changes.

Cover a synthetic coast/threat bottleneck requiring initial movement away, end-to-end checked legs, captured threat/terrain hints, and enclosed/unavailable/invalid conditions. Expose guidance and node-budget exhaustion in route diagnostics. Inspect any remaining live recovery budget incompatibility separately rather than silently broadening an order. After focused checks and the full gate, deploy the native build and attempt one guarded recovery with retained evidence. Resume positive food delivery and repair only after a verified safe home return.

## Notes

**agent:codex/renki-milestones** at 2026-11-01T00:32:32Z

Implemented ordinary physical search followed by at most one threat-guided fallback. Guidance uses a bounded visibility graph and terrain hints, with sticky preference only during the same movement goal; actual support, water, body, floor, obstacle and refreshed hostile probes remain authoritative. No recovery limits or clearances changed. Diagnostics expose phase, node exhaustion, hint counts and graph build time.

`just check plugin test-core` passed all 679 non-interop checks; `just check plugin build` passed with the game assemblies available. ThreatGuidanceTests prove initially-away progress through checked legs to actual arrival, ordinary-search bridge preservation, bounded unchanged failure, changed live-threat refusal and wrong-floor/unavailable-support refusal. The captured terrain fixture proves only a geometric alternative, not a physically traversable live route. Corrected the synthetic test to retain the same sticky guidance preference as the native walker and use each search's actual progress metric.

Full `just check ci` is running. Live guarded recovery remains unverified; Tas is offline with the panel online. A long geometric alternative may exceed the existing recovery limits; retain those limits and inspect the observed outcome rather than broadening the order silently.

**agent:codex/renki-milestones** at 2026-11-01T00:38:32Z

Live guarded recovery passed on deployed29db3b8: `.checks/recovery-20261101T003515Z`, full audit `.checks/audit-window-GWZnrB.jsonl`, 00:35:15Z–00:37:44Z. Tas returned from[-23.24,30.87,-30.42] to[-37.02,36.09,9.11], with grounded/in_shelter/near_fire observed and70.4/70.4 HP at release. About110.7m observed travel and100.5s in-world, inside the unchanged160m/120s/80m bounds. No injury was observed; HP/max declined together as food aged. The correct home floor and shelter were confirmed by the native guard and independent supervisor.

The live fallback was exercised: at00:36:18Z, the ordinary search exhausted4096 nodes and the guided phase found an initially-away checked leg after103 more expansions (4199 total), ending near[-19.02,36.21,-48.36]. Guidance used16 ring points/602 terrain hints, estimated234.23m initially, and built in13.65ms. Total search work1151.57ms over11.02s; worst measured slice15.01ms, including the synchronous graph build. Subsequent fresh-threat guidance became inactive as observations changed, and ordinary physical searches completed the return. The captured static geometric estimate is not a prediction of actual travelled distance around moving creatures. Repeated entrance replans were still needed before the west stair approach succeeded.

`just check ci` passed in `.checks/20261101T003211Z-ZOPLmM.log`; committed `just check review t3/outdoor-household-validation..HEAD` passed with no findings in `.checks/20261101T003442Z-b0oYp3.log`. `just check vm install` verified installed DLL/package checksums. PR #138 https://github.com/warricksothr/renki/pull/138 stacks on #137. Outdoor food delivery may now be retried; repair remains a separate live check. A one-time13.65ms graph-build spike is retained as a performance limitation, not presented as meeting the2ms slice target.

## Summary

Implemented in29db3b8 / PR #138; guarded live return exercised guided fallback and reached home floor/shelter without injury. Evidence and measured13.65ms synchronous graph-build limitation are in the latest note.
