---
schema: 4
id: TKT-01M6KXKRTK0CXTWC4KV2YGC5TX
title: Protect base resources and reject underwater harvest sites
type: bug
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - safety
  - controller
  - plugin
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
  actor: agent:codex/harvest-zones
  branch: t3/renki-portal-travel-startup
  worktree: /home/sothr/.t3/worktrees/renki/t3-6c0da0c0
  commit: 7c67dfa7c23ceedd83b17e901799ce49c7e4a4ad
  session: null
  claimed_at: 2026-11-04T18:37:41Z
  expires_at: null
archive: null
created_at: 2026-11-04T18:37:17Z
updated_at: 2026-11-04T18:51:00Z
created_by:
  id: agent:codex/harvest-zones
  name: ""
updated_by:
  id: agent:codex/harvest-zones
  name: ""
extensions: {}
---

## Description

The owner observed new wood map locations in water and requested protection for decorative trees and rocks throughout camp. The owner selected the saved 60 m new home base boundary, centered at [-204.39,32.97,-467.43]. Survey currently records resources inside that boundary, including FirTree_oldLog roots at Y25–26 below water. Background discovery has the same missing filters.

Implement saved-base protection in discovery, gather selection and task dispatch/native harvesting. Remove invalid automatic resource markers without altering owner-managed places, role bindings or deletion suppression. Reject underwater resources and avoid treating fallen static logs as standing renewable wood sites. Keep Tas and private video online outside the required deployment restart; verify refusal without damaging camp resources.

## Acceptance criteria

- [x] Saved base circles exclude wood and stone discovery and actual harvesting, including their edges and overlapping work zones.
- [x] Underwater roots and fallen static logs do not become automatic wood sites; unknown water evidence is handled conservatively.
- [x] Invalid automatic resource markers are pruned without changing owner places, bound roles or suppression; valid outside sites remain discoverable.
- [ ] Focused regressions and full CI pass; deployed runtime safely refuses a camp target and invalid new-home automatic markers remain absent.

## Implementation plan

Use every saved base circle in the current world as a no-harvest exclusion; the selected new home circle remains exactly 60 m. Share resource-site eligibility across background and survey discovery. Native perception supplies measured water level at the resource root; unknown/nonfinite or submerged roots cannot establish automatic wood/stone/ore sites. Classify static fallen logs separately from standing trees.

Prune only automatic wood/stone/ore markers with positive invalid evidence (inside a saved base, observed wet root, or fallen-log wood evidence). Preserve owner edits, all saved-base role bindings, unrelated sites and suppression. Recheck eligibility transactionally during both discovery store writes.

Gather selection checks saved exclusions before travel. Every application harvest dispatch resolves a fresh current-world observed entity, rejects protected/wet targets and supplies authoritative bounded exclusions. A new advertised native feature prevents an older plugin silently ignoring those exclusions. Native harvesting checks the actual target root against the supplied circles and current water before work and each tick, retaining existing collision and structure guards.

Add regressions for captured bad positions, edge/overlap targets, unknown water, outside sites, owner/role/suppression protection, protocol validation and old-plugin refusal. Run focused checks, full CI, commit and committed review, then deploy with the VM lease. Verify safe camp refusal and marker cleanup without striking resources; restore Tas/private output online and update the handoff.

## Notes

**agent:codex/harvest-zones** at 2026-11-04T18:48:14Z

Implemented saved-base protection across both discovery paths, gather selection, every application harvest dispatch, and native task start/ticks. The selected new home circle remains exactly 60 m; all saved circles in the current world protect resource roots at all heights. Native capability harvest_exclusions_v1 prevents silent fallback to an older plugin. Resource-root water evidence rejects wet/unknown automatic wood/stone/ore sites, and FirTree_oldLog is classified as log rather than standing tree. Existing native structure/fall/sight/ward guards remain.

Focused home/app/behaviour tests passed; protocol/session checks passed after matching strict exclusion decoding to the native decoder (.checks/20261104T184316Z-Ua62JN.log). Native Harvest checks passed (.checks/20261104T184224Z-9FPIkr.log). Initial full CI hit an existing TestKnownThreatTriggersRetreatThroughBridge/known_pursuit 2 s timeout, with no race report; five focused race runs passed (.checks/20261104T184712Z-Brbyrw.log). Full gate rerun is pending. Do not count the first gate as a pass.

Tas stayed online during coding. Before deployment, ordinary t-29 reached the food barrel approach. t-30 partially supplied 3 BoarJerky and 3 QueensJam but could not supply missing Sausages (.checks/task-gzu65aif.json). Fresh barrel contents showed DeerStew; t-32 withdrew exactly 3 (.checks/task-s33_5u2w.json). Ordinary t-34 returned to the sheltered eastern workbench (.checks/task-mpihgx1t.json). No harvesting was requested or done. VM lease is held as agent:codex/harvest-zones for deployment and safe refusal checks.

**agent:codex/harvest-zones** at 2026-11-04T18:51:00Z

Full just check ci passed in 186 s, including the actual Valheim plugin build and all protocol/native checks (.checks/20261104T184737Z-zO9WFA.log). Criteria 1–3 are earned by the implementation and regressions. Criterion 4 still awaits committed review, installation and safe live camp refusal/map cleanup checks. No camp resource will be struck to validate this guard.
