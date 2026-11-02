---
schema: 4
id: TKT-01M6EYEMTXMJN0GSF2Y03CDB12
title: "Panel: show loot's anchor, settle area, reach and drop decisions"
type: task
status: draft
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
updated_at: 2026-11-02T20:15:45Z
created_by:
  id: agent:claude-code/renki-wave1
  name: ""
updated_by:
  id: agent:claude-code/renki-wave1
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

- [ ] The panel map draws the anchor, settle area and current reach live
- [ ] Drops loot would take, ignores as deliberate and cannot reach are marked differently, in a panel UI scenario
- [ ] The loot card shows settle time, current reach and the parameters
- [ ] A live session under the VM lease records the reach over time in a note
