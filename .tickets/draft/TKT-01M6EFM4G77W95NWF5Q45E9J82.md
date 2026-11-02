---
schema: 4
id: TKT-01M6EFM4G77W95NWF5Q45E9J82
title: "VM: two 2 h base upkeep soaks with the upkeep report"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - vm
  - tooling
  - live
assignees: []
milestone: M5
parent: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
origin: null
dependencies:
  - TKT-01M6EFK45386G1K5JCGB4550MB
  - TKT-01M6DDFTF6N15TE3ANQ5KWT1Q7
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
  - TKT-01M62VXZ5PEV9CNPVYHC5TTF09
  - TKT-01M62VXZ6ZWJFZ5X6JJX0WGDCX
  - TKT-01M62VXZ84SA9TSJC6NJNPV8KJ
  - TKT-01M6EFKDSV1BVWR6YYJF6KSZRA
  - TKT-01M6AT4EFWX7YMRCJS8MX3SV32
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:56:36Z
updated_at: 2026-11-02T15:56:36Z
created_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
extensions: {}
---

## Description

M5's base-upkeep goals (docs/ROADMAP.md, M5 goals 5.1, 5.2, 5.3 and 5.6) are measured over two consecutive 2 h home soaks. M4 builds the soak supervisor `just vm soak HOURS` (TKT-01M6DDFTF6 (VM: 24 h soak with crash, leak, growth and store checks)), which watches for crashes, leaks and store errors. This ticket adds an upkeep profile to it and runs the M5 soaks.

### What to build

- A `--upkeep` option on `just vm soak` that, before the soak, checks that the owner's saved upkeep instances exist and are enabled (fuel from TKT-01M62VXZ5P, tend from TKT-01M62VXZ6Z, chest assignments from TKT-01M62VXZ84 and sorting from TKT-01M6EFKDSV1B, household food from TKT-01M6AT4EFW). It never creates or enables instances on its own; a missing one fails with its name. Afterwards it restores nothing it did not change.
- At the end, it runs `just vm upkeep-report` (TKT-01M6EFK45386 (VM: base upkeep report from a home soak audit window)) over the soak window and folds the report's section verdicts into the soak's `summary.json`.
- The soak keeps M4's failure conditions and cleanup. Combat stays off.

### Files

- `scripts/vm/vm.just`: the `soak` recipe from TKT-01M6DDFTF6
- the soak supervisor script that ticket adds under `scripts/vm/`
- `docs/TESTING.md`: how to run an upkeep soak

### Verification

Offline: the soak's self-test (wired into `just ci` by TKT-01M6DDFTF6) gains cases for a missing upkeep instance and for folding a failing report section into the verdict. `just check ci`.

Live: `just vm soak 2 --upkeep` twice in a row, holding the VM lease (TKT-01M6EF5DTG). Each `summary.json` shows the soak verdict `pass` and the report sections for fires, cooking, assignments and pantry `pass`. Link both paths in a note here and on the goal tickets.

## Acceptance criteria

- [ ] just vm soak HOURS --upkeep fails with the instance's name when a required upkeep instance is missing or disabled, and never creates or enables one
- [ ] The soak's summary.json carries the upkeep report's section verdicts and fails when any section fails
- [ ] The soak self-test in just ci covers a missing instance and a failing report section
- [ ] Two consecutive just vm soak 2 --upkeep runs pass, with both summary.json paths linked here and on TKT-01M62VXZ2F
