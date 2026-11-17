---
schema: 4
id: TKT-01M6KX627XMJJWVCFB4VWCFXAT
title: "Repair: approach inside the station's native use radius"
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - movement
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
created_at: 2026-11-04T18:29:47Z
updated_at: 2026-11-04T18:29:47Z
created_by:
  id: agent:codex/gear-up
  name: ""
updated_by:
  id: agent:codex/gear-up
  name: ""
extensions: {}
---

## Description

### Observed failure

During owner-requested warehouse repair, t-22 interact(repair:true) reached eastern workbench1:15887 at[-158.49,30.30,-476.39] but returned refused: the station refused use, without an ordinary roof/fire message. Tas stopped at[-159.27,30.97,-474.67], approximately2.004m in3D from the station. RepairTask uses a fixed StopDistance2 with a walker that accepts planar arrival plus a separate vertical tolerance. Native CraftingStation.InUseDistance instead requires strict Vector3.Distance < m_useDistance (2m here). Thus a successful approach can still leave the body outside normal use range on a raised floor.

### Expected correction

Approach through supported clear footing inside the station's actual finite native use radius with a small margin, checking real3D distance and live sight/ward access. Reuse bounded alternate interaction approaches when the station center is an unsuitable standing goal. Do not widen the game's range, bypass station type/level/roof/fire rules, edit durability, or blind-retry an ambiguous use. Preserve native inventory durability receipts and report station identity and observed level to distinguish positioning from repair eligibility.

### Native evidence

An explicit ordinary move t-24 to[-159.05,30.97,-475.10] ended at[-159.00,30.97,-475.07],inside the same native use radius. Repeating the normal repair task t-25 at this changed standing position succeeded:3 carried items repaired, leather helmet393.4->400,chest394.4->400,legs236.1->400. AxeFlint remained100/100; injurySeq0. This is a position correction, not relaxation of native rules or a source-code fix. The owner's west forge is1:15324[-165.67,30.31,-473.28]; eastern workbench1:15887 is the station that actually repaired this gear. Exact station levels were not exposed, so do not invent them.

Evidence .checks/task-gyvdqj9w.json,.checks/warehouse-workbench-refusal-status.json,.checks/task-5imt_75g.json,.checks/task-skao5zl2.json,.checks/warehouse-repair-{status,inventory}.json. Installed plugin/controller1c1d36c.

### Scope

File as draft for owner selection. Today's live request was fulfilled by ordinary closer standing. No code change or redeployment was needed. Do not promote this implementation follow-up automatically.

## Acceptance criteria

- [ ] Approach uses the actual native use distance and3D geometry with a margin on raised floors
- [ ] Core regression reproduces the captured planar-arrival/native-distance mismatch
- [ ] Live repair reaches usable footing and verifies durability gains under normal station rules
