---
schema: 4
id: TKT-01M6F877XVE8DPB4WA9WXQVJKE
title: "Behaviours: needs ledger ranks passive work by what the base lacks"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M5
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies:
  - TKT-01M6EWN4WH7R17QC51VKH4NDYW
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T23:06:28Z
updated_at: 2026-11-02T23:06:37Z
created_by:
  id: agent:claude-code/renki-lead
  name: ""
updated_by:
  id: agent:claude-code/renki-lead
  name: ""
extensions: {}
---

## Description

Part of the idle/passive decision (note on TKT-01M62VXYQB, 2026-11-02). The controller should know what the base is short of, so that passive work can prefer it inside the settle area. Needs re-rank nearby work; they never send Tas away.

### What to build

- A ledger of wanted items (food reserves such as berries and mushrooms, building stock such as stone and wood), each with a target and the current count.
- The counts come from what the controller already knows: memory of chests, the inventory, and the pantry/food rules (docs/FOOD.md, docs/MEMORY.md).
- Passive list entries can target "wanted" items, so that picking raspberries in the area ranks above mining stone when food is low.
- Owners can see and set the targets from the panel. The agent side reads the ledger.

### Verification

- Tests: counts from memory and inventory, re-ranking of passive work by want, and no work chosen outside the area.
- `just check ci`.

## Acceptance criteria

- [ ] A ledger of wanted items with targets and counts from memory and inventory, in tests
- [ ] Passive entries can target wanted items, re-ranking work inside the settle area only, in tests
- [ ] Owners see and set targets in the panel
