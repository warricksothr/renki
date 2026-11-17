---
schema: 4
id: TKT-01M6EYEMTXMJN0GSF2Y03CDB12
title: "Panel: show loot's anchor, settle area, reach and drop decisions"
type: task
status: review
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - live
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M6EYEMRVAG415PX2A10XSXDC
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T20:15:45Z
updated_at: 2026-11-04T03:38:15Z
created_by:
  id: agent:claude-code/renki-wave1
  name: ""
updated_by:
  id: agent:codex/portal-deploy
  name: ""
extensions: {}
---

## Description

The adaptive loot reach (settle area plus a growing radius) has several numbers that can only be tuned by watching them in play. Show them on the panel so the owner can see why Tas did or did not go for a drop.

### What to build

- **Map layer.** On the panel map, draw the anchor player, their settle area and loot's current reach, updating live.
- **Drops.** Mark the drops loot would take, the ones it is ignoring as deliberate, and the ones out of reach, each in its own style.
- **Behaviour card.** The loot behaviour's card shows the settle time, the current reach, and the parameters with their defaults.
- **Controller.** The controller exposes what the layer needs through the existing state the panel already polls. Add no new endpoint unless that state can't carry it.

### Files

- `internal/behaviour/loot.go` (exported status), `internal/panel` (state the panel reads)
- `panel/src/` (map layer and behaviour card), plus a panel UI scenario in the existing suite

### Verification

- A panel UI scenario with a settled anchor, a reach circle and the three kinds of drop marker. `just check panel test`, then `just check ci`.
- Live (label `live`, under the VM lease): watch the layer while an owner stands, fells a tree and walks off, and record the reach over time in a note.

## Acceptance criteria

- [x] The panel map draws the anchor, settle area and current reach live
- [x] Drops loot would take, ignores as deliberate and cannot reach are marked differently, in a panel UI scenario
- [x] The loot card shows settle time, current reach and the parameters
- [ ] A live session under the VM lease records the reach over time in a note

## Implementation plan

Expose per-instance Loot diagnostics through existing behaviour status: current anchor and scope/freshness, settle centre/radius/time, adaptive reach, effective parameters/defaults and each visible drop's decision. Reuse the actual pickup eligibility policy so the explanation and selection agree; keep disabled state explicit. Add a selectable read-only Loot card and map layer for anchor, settle area, reach and differently styled drop decisions. Expire overlays on offline/stale/world/connection changes and test via existing panel unit/UI scenarios on desktop and mobile. Run just check panel test, focused panel UI checks and just check ci. Leave the owner-in-game reach-over-time criterion open; keep production Loot disabled and the game offline.

## Notes

**agent:codex/portal-deploy** at 2026-11-03T18:05:28Z

### Read-only Loot diagnostics and map implemented

Implemented on t3/renki-runtime-loot-batch. The existing /api/behaviours status now carries per-instance Loot diagnostics with connection/world/UID, frame timestamps/sequence, anchor, settle centre/radius/duration, current reach, effective parameters/defaults and visible drop decisions. Pickup selection and explanation use one decision function. The new card shows enablement, inferred nearby work and drop reasons; choose a Loot instance when several exist. The map layer draws a square anchor, dashed settle circle, solid reach and distinct filled/cross/hollow eligible/deliberate/out-of-reach markers. Other exclusion reasons remain visible in the card. Disabled instances offer a policy preview without becoming enabled.

The overlay expires on stale/offline/scope changes and saved-atlas views. Polling an unchanged sequence/frame cannot refresh its lifetime; stalled requests/streams still expire it. just check panel test passed (.checks/20261103T180016Z-6oRuGE.log); the desktop/mobile loot-reach UI scenario passed (.checks/20261103T180124Z-uOB6Gp.log), checking card/defaults, real canvas colors, reach/settle export, toggling, expiry and absence of non-GET requests. Screenshots were inspected. Focused Go/race tests prove decisions match pickup selection and stale diagnostics clear. Full just check ci remains required before commit.

Criterion 4 stays unchecked: the owner explicitly requires keeping Valheim offline. No production setting or binary was changed by this implementation; Loot remains disabled. After a safe supervised rejoin, watch the layer while the owner stands, fells and walks away and record actual reach over time. This is separate from portal following's still-unearned native crossing check.

**agent:codex/portal-deploy** at 2026-11-03T18:09:32Z

The complete just check ci gate passed (225 s), retained in .checks/20261103T180530Z-ShHgUG.log. This covers the final activity/diagnostics source, panel unit/UI suites, controller race tests/native builds, plugin/interop and deployment entry-point checks. Committing the batch before committed review; production remains unchanged and the game offline.

**agent:codex/portal-deploy** at 2026-11-04T03:38:15Z

### Merged batch deployed and Tas brought online

The owner explicitly requested deployment and bringing Tas online, superseding the implementation-only offline instruction. PRs #176, #177 and #178 are merged; deployed main 501ca8d under the VM lease using just check vm install (24 s; .checks/20261104T033528Z-rRGmvh.log). Native compatibility preflight, installed checksums and deployment receipt verified; durable data retained. Started the panel on the new version, then just check vm start passed (64 s; .checks/20261104T033606Z-hUky0q.log).

Fresh 2026-11-04 03:37Z observations: Tasleroch connected in DizzysHellheim at [-31.08,36.49,15.19], grounded/sheltered, 25/25 HP, 50/50 stamina, no recorded injury. Inventory and active/carried meals are empty. One boar about 30 m away had no observed targeting of Tas; no players or Greylings were observed. Saved home is about 513 m away. No travel/recovery task was ordered. Loot remains disabled, not running and not pending; the new diagnostics are fresh. Combat remains allowed; RecoveryGuard and AutoLogout remain false. Valheim PID 36840 and controller PID 31360 running. Leave Tas online as requested, and stage food/gear before travel or native portal/Loot trials. This deployment does not earn any remaining live acceptance criterion.

Evidence: .checks/deploy-online-observation.json, .checks/deploy-online-map.json, .checks/deploy-online-inventory.json, .checks/deploy-online-behaviours.json. docs/HANDOFF.md updated with current merged/deployed state and historical observations retained separately.

## Summary

Implemented scoped Loot diagnostics, a read-only card and map layer using shared pickup decisions. Unit, desktop/mobile UI, race and full CI passed. Live reach-over-time observation remains unchecked while the game stays offline.
