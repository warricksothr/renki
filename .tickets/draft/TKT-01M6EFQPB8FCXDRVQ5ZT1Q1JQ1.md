---
schema: 4
id: TKT-01M6EFQPB8FCXDRVQ5ZT1Q1JQ1
title: Earn three consecutive bounded expedition passes on the VM
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - vm
  - live
  - movement
assignees: []
milestone: M7
parent: TKT-01M66PMASYEM3Z9KJMB2979CEM
origin: null
dependencies:
  - TKT-01M6EFQP437YT124TQ25299BJV
  - TKT-01M6EFQP5XBXHS67Q03PXCZPT0
  - TKT-01M6EFQP851CG8DV29FSHZ7KMP
  - TKT-01M6EFQNWT6P9JJA3KVHFSQA2F
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:58:33Z
updated_at: 2026-11-02T15:58:44Z
created_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
extensions: {}
---

## Description

The M7 benchmark. With the explore goal (TKT-01M6EFQP43 (Explore goal runner: legs, discovery and return proof)) and the supervisor (TKT-01M6EFQP85 (VM: expedition-trial supervisor with coverage and return evidence)) deployed, run `just vm expedition-trial` from an owner-approved dry-ground anchor in a familiar biome, supplied with food and gear, combat off, until three consecutive runs pass. Also run it once with `--stop-after 60`. Record each summary.json path in a note.

### Pass rule per run (goal 7.3)
- radius at most 150 m, elapsed budget 10 min, work budget 5 min, travel at most 600 m
- at least 2,000 m2 (125 canonical 4 m samples) of new measured terrain inside the circle
- succeeded return receipt and a later state within 3 m horizontally and 1.75 m vertically of the pinned anchor
- no death, combat off before and after

A run where no safe frontier remains is an honest partial and breaks the streak; pick a new anchor rather than widening the radius.

### Files
- scripts/vm/vm.just (recipe from TKT-01M6EFQP85), docs/TESTING.md, docs/EXPLORATION.md (record the chosen anchor and numbers)

### Verification
- `just vm deploy` then `just vm expedition-trial ...` (takes the VM lease); evidence is `.checks/expedition-trial-<ts>/summary.json`

## Acceptance criteria

- [ ] Three consecutive `just vm expedition-trial` runs pass by the rule above; their summary.json paths are in a note
- [ ] One run with `--stop-after 60` reaches idle within 2 s of the stop, cancels the goal and sends no further task; its summary.json path is in a note
- [ ] Every run, passing or not, ends with combat off and the goal cancelled or terminal
