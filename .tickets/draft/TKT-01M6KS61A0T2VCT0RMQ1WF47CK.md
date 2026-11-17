---
schema: 4
id: TKT-01M6KS61A0T2VCT0RMQ1WF47CK
title: Build and claim a bed through native placement
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
  - live
assignees: []
milestone: null
parent: TKT-01M6CEA0K3Q2V6QYWZBRN8T5ES
origin: null
dependencies:
  - TKT-01M6CEA0MRCXAD97ZXSK43H3EG
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-04T17:19:52Z
updated_at: 2026-11-04T17:19:52Z
created_by:
  id: agent:codex/portal-deploy
  name: ""
updated_by:
  id: agent:codex/portal-deploy
  name: ""
extensions: {}
---

## Description

The owner requested a ticket for a Renki agent to create a bed after Tas reached the relocated home and all four freshly observed beds were owned by other characters. A historical lower-floor bed marker was no longer observed; the owner thinks they moved that bed and will place a new one for the current session. This ticket covers future construction capability; it does not authorize replacing or taking another character's bed.

### Bounded first construction use case

Accept an owner request to build one ordinary bed at a selected location near a configured base. Reuse the native placement foundation in TKT-01M6CEA0MRCXAD97ZXSK43H3EG (Add bounded earthwork and building actions with plan previews). Resolve the current runtime's bed recipe, hammer, durability, station and material prerequisites rather than assuming creative availability or hard-coding mod-dependent requirements. Report missing prerequisites before mutation; automatic harvesting, withdrawing supplies, making tools, adding shelter/fire and demolishing existing pieces are separate work.

Preview the bed footprint, orientation, actual floor height, support and a reachable standing/use approach. Respect wards, collision, existing furniture, doors and access paths, including stacked floors. Place through the normal native game action with its checks and actual resource/tool costs. Do not instantiate a prefab directly, spawn resources or bypass placement validation.

### Verified result and interruption

Observe the newly placed bed's world/entity identity and transform, verify that it persists and can be approached, then claim/set spawn through the existing native bed interaction. Verify ownership belongs to Tas and the spawn result before reporting success or binding the base sleeping place. Distinguish placement, ownership, spawn binding and actually sleeping; report unmet roof/fire/enemy/wetness or other native conditions without claiming that placing a bed guarantees usable rest. Never change home, claim another player's bed or start sleep merely as a side effect of construction.

Stop on cancellation, takeover, threats, injury, loss of support or changed placement scope. Retain enough attempt evidence to reconcile observed pieces and material changes after a timeout/reconnect; uncertain completion must not place another bed on retry. Invalidate affected local geometry/map observations and retain protected owner markers.

### Validation

Add focused policy/protocol/controller regressions and named just recipes for repeatable checks as needed. Run just check ci and committed just check review. First native placement checks belong in an isolated authorized test plot; a shared-base trial requires a specific owner-selected site and the VM lease. Keep actual native placement/claim evidence separate from offline checks.

## Acceptance criteria

- [ ] An owner can select a bounded bed site and inspect floor-aware placement, access and native tool/material/station prerequisites before construction.
- [ ] One bed is placed through ordinary native game checks; observed entity/transform and material/tool effects prove placement without spawned supplies or damage to existing pieces.
- [ ] The agent verifies safe reach, claims only the newly observed unowned bed, confirms its own ownership/spawn result, and updates the selected base sleeping approach only after confirmation.
- [ ] Missing prerequisites and native rest/claim refusals are explicit; placement, claim/spawn and sleeping remain distinct outcomes.
- [ ] Cancellation, takeover, changed scope and uncertain interruption stop or reconcile work without duplicate beds, unrequested demolition or another player's bed claim.
- [ ] Focused checks, just check ci and committed just check review pass; a VM-leased isolated native trial retains actual placement, resource and ownership evidence.
