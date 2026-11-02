---
schema: 4
id: TKT-01M6EFKWEDS35J58RCB7YEH71Y
title: "VM: gather-trial --gather-cycles evidence for repeated work cycles"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - vm
  - live
assignees: []
milestone: M5
parent: TKT-01M6864352N8E28YCGHA19AN90
origin: null
dependencies:
  - TKT-01M6EFKWBXB8D3XZ7G2T4E651W
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
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

M5 goal 5.7 (docs/ROADMAP.md) needs live evidence of repeated work cycles. `just vm gather-trial` (scripts/vm/combat-trial.py with `--gather-tree`) proves one trip of up to three trees. This ticket adds a cycles option to the trial and runs it.

### What to build

- `--gather-cycles N` (1 to 4) on the gather trial. It starts the work-cycle instance from the controller ticket filed with this one and requires, per cycle: at least one complete tree with task-matched break proof, a home deposit with matched receipts, a recorded pressure reason for the return, and arrival back within 12 m of the same site before the next cycle. It also requires a repair receipt when the axe started a cycle below 90 %.
- The trial's `summary.json` lists each cycle with its pressure inputs, reason, deposit receipts and duration.
- The offline self-test (`scripts/vm/gather-trial-test.py`) covers a passing two-cycle run, a cycle that returned without a pressure reason, and a second cycle at a different site.

### Files

- `scripts/vm/combat-trial.py`: the `--gather-*` options
- `scripts/vm/gather_trial.py` and `scripts/vm/gather-trial-test.py`
- `scripts/vm/vm.just`: `gather-trial`, `test-gather-trial`

### Verification

Offline: `just check vm test-gather-trial` and `just check ci`.

Live, holding the VM lease (TKT-01M6EF5DTG): `just vm gather-trial --gather-cycles 2` passes 2 runs in a row, each under 30 minutes, with no death. Link both `summary.json` paths here and on TKT-01M6864352.

## Acceptance criteria

- [ ] --gather-cycles N requires per cycle a complete tree, a matched home deposit, a pressure reason and a return to within 12 m of the same site
- [ ] just vm test-gather-trial covers a passing two-cycle run, a return with no pressure reason, and a second cycle at another site
- [ ] Two consecutive just vm gather-trial --gather-cycles 2 runs pass, each under 30 min with no death, with summary.json paths linked
