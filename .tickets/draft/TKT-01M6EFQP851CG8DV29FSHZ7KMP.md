---
schema: 4
id: TKT-01M6EFQP851CG8DV29FSHZ7KMP
title: "VM: expedition-trial supervisor with coverage and return evidence"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - vm
  - tooling
assignees: []
milestone: M7
parent: TKT-01M66PMASYEM3Z9KJMB2979CEM
origin: null
dependencies:
  - TKT-01M6EFQP110HAMEANK85XT7F5J
  - TKT-01M6EFQN74X0EDYJAVBJJK2DY8
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

There is no recipe that measures an expedition. Add `just vm expedition-trial` (scripts/vm/vm.just, scripts/vm/expedition-trial.py) and its offline test recipe `just vm test-expedition-trial` (scripts/vm/expedition-trial-test.py), wired into the root `just ci` list beside the other `vm::test-*` recipes.

Follow scripts/vm/combat-trial.py `--persistent` (goal-trial) and scripts/vm/route-benchmark.py. Steps: take the VM lease once TKT-01M6EF5DTGGAB15GFMRMDT8KTA lands; preflight (live identified world, combat off, at the anchor); record the coverage baseline (`GET /api/atlas/coverage` with `since` = now) and an atlas-check style viewport; POST the explore goal; poll goal and status with a wall-clock cap of the goal's elapsed budget plus 2 minutes; at the end read coverage since start, the arrival evidence and an audit window; clean up (cancel the goal, keep combat off, stay online unless `--allow-logout`), then release the lease.

`summary.json` records: pass, outcome, new_m2, required_m2, radius, elapsed_s, work_s, travel_m, arrival (flat_m, dy_m, receipt task id), deaths, combat setting before and after, stop latency if used, and evidence paths. `pass` is true only for outcome succeeded, new_m2 at or above required, arrival within 3 m and 1.75 m, elapsed within budget and no death. `--stop-after SECONDS` sends a chat stop mid-run and records time to idle.

### Files
- scripts/vm/vm.just, scripts/vm/combat-trial.py, scripts/vm/route-benchmark.py, scripts/vm/atlas-check.py (sample extraction to reuse)
- justfile (root `ci` list), docs/TESTING.md

### Verification
- `just check vm test-expedition-trial`
- `just check ci`

## Acceptance criteria

- [ ] `just vm expedition-trial` refuses to start without a live identified world, with combat on, or away from the anchor, and never enables combat
- [ ] summary.json has the fields above, and `pass` follows the rule above (tested)
- [ ] `--stop-after SECONDS` sends a chat stop and records the time to the first idle state
- [ ] Cleanup always cancels the goal, leaves combat off and leaves the panel available, including after an exception (tested)
- [ ] `just vm test-expedition-trial` covers pass, partial, death, timeout, stop and refused preflight from recorded fake responses, and runs in `just ci`
- [ ] docs/TESTING.md lists the recipe and its pass rule
