---
schema: 4
id: TKT-01M6EFKWBXB8D3XZ7G2T4E651W
title: "Controller: gather work cycles that deposit and resume the site"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M5
parent: TKT-01M6864352N8E28YCGHA19AN90
origin: null
dependencies:
  - TKT-01M62N12JW8TFN12MAT8M1ADZ8
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:56:28Z
updated_at: 2026-11-02T15:56:28Z
created_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
extensions: {}
---

## Description

M5 goal 5.7 (docs/ROADMAP.md): production work continues across deposits instead of ending after one trip. The `gather` template already has the first bounded pressure controls (docs/BEHAVIOURS.md, "Bounded beech gathering"): between trees, capacity or axe-durability pressure, low stamina after one rest, or less than four minutes left chooses delivery, and `returnHome` repairs the axe at a home workbench before leaving. But a run is at most three trees and ends at home. The epic TKT-01M6864352 (Pressure-driven work, storage and maintenance goals) asks for a work cycle that deposits and then goes back to the remembered site.

### What to build

A `cycles` parameter on `gather` (default 1, at most 4), or a thin `work_cycle` template wrapping it, that:

- Runs one gather trip, delivers home through the existing `returnHome` path, then repairs a carried axe below 90 % durability at the home workbench (the existing `interact` with `repair:true` and its fresh-durability receipt), checks food and health with the same admission as a new trip, and travels back to the same remembered site for the next cycle.
- Records, per transition, the pressure inputs (free slots, carry fraction, axe durability, stamina, food time left, time left in the budget) and the reason it chose to return or continue. These go into behaviour status and an audit log record.
- Uses hysteresis: once a return is chosen it is committed until the home floor is confirmed, as household food does (docs/FOOD.md, "Desired pantry outcomes").
- Stops, rather than loops, when the site has no safe trees, a repair fails, food is short, or an owner order, retreat or session change revokes the lease. Each stop names its reason.
- Keeps all of gather's existing bounds per cycle, plus an overall limit of 40 minutes.

Armour repair, cargo value and storage other than home stay with the epic and TKT-01M64T4WG (Tools: acquire, craft, repair and upgrade harvesting equipment).

### Files

- `internal/behaviour/gather.go`: the pressure check between trees (search for `errGatherCapacity`)
- `internal/behaviour/wood.go`: the `go chop wood` instance and saved site
- `internal/behaviour/health.go`: admission checks
- `internal/goal/food_work.go`: an example of a committed return phase with hysteresis
- `docs/BEHAVIOURS.md`: document the cycles

### Verification

`just check controller test-packages ./internal/behaviour` with tests for: two cycles back to the same site, a return chosen by capacity and one by durability, the committed return not reversed by a small inventory change, a failed repair stopping before the second cycle, and an owner stop mid-cycle. `just check ci`. Live evidence is the gather-trial ticket filed with this one.

## Acceptance criteria

- [ ] A gather instance with cycles 2 returns home, deposits, repairs a worn axe and goes back to the same remembered site, in a test
- [ ] Each return records its pressure inputs and reason in behaviour status and the audit
- [ ] A committed return is not reversed by a small inventory change, in a test
- [ ] A failed repair, missing food, no safe trees or an owner stop ends the run with a named reason instead of looping, in tests
