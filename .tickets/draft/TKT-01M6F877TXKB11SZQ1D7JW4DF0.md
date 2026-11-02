---
schema: 4
id: TKT-01M6F877TXKB11SZQ1D7JW4DF0
title: "Behaviours: passive gather works alongside the anchor"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M6EWN4WH7R17QC51VKH4NDYW
  - TKT-01M6F877Q8WJ2YPMG4JMY9HYCV
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

Part of the idle/passive decision (note on TKT-01M62VXYQB, 2026-11-02). When the anchor player stops to work, Tas should work with them: harvesting what they harvest, where they are.

### What to build

- A built-in passive instance, **on by default** (the owner's choice), built on the `gather` template.
- Its target comes from the anchor's activity: felling beech means beech trees, mining copper means copper.
- It works only inside the anchor's settle area and loot's reach.
- It sits in the built-in passive list after loot, so drops are collected first and loot picks up what both of them knock down.
- It never fells or mines in the home area or on structures. It keeps clear of the anchor's own target, so they do not both swing at one tree.

### Verification

- Tests: the activity picks the target, the area bounds it, and Tas avoids the anchor's own target.
- `just check ci`. Live, under the VM lease: fell beech beside Tas while it follows you, and watch it fell nearby beech and collect the wood.

## Acceptance criteria

- [ ] A built-in passive gather instance, on by default, harvests what the anchor's activity names, inside their settle area, in tests
- [ ] It follows loot in the built-in passive list and keeps off the anchor's own target
- [ ] Live, under the VM lease: Tas fells beside an owner felling beech and the wood is collected
