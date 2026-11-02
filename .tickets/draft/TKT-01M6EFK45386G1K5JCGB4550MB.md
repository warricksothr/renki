---
schema: 4
id: TKT-01M6EFK45386G1K5JCGB4550MB
title: "VM: base upkeep report from a home soak audit window"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - vm
  - tooling
assignees: []
milestone: M5
parent: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:56:03Z
updated_at: 2026-11-02T15:56:03Z
created_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
extensions: {}
---

## Description

M5 (runs the base) is measured over 2 h home soaks: fires stay lit, no food burns, assigned items reach their chests, and the pantry stays stocked (docs/ROADMAP.md, M5 goals 5.1, 5.2, 5.3 and 5.6). Nobody can watch two hours of play, so the numbers have to come out of the audit log afterwards. No recipe computes them today. This ticket builds that report. It is offline tooling: it reads a saved audit window and needs no game.

### What the audit already holds

The audit (docs/AUDIT.md) keeps every `perception` frame (1 Hz) and `state` frame (5 Hz). Perception structures already carry `fire: {fuel, fuelMax, lit}` for fires and `cooking: {heated, supported, slots:[{index,item,state,elapsedS}]}` for cooking stations (docs/BRIDGE-PROTOCOL.md, structures). State carries `time.night`, `pos`, `effects` and the task. `container` frames carry chest snapshots, and `task` frames carry deposit and withdraw receipts. Goal summaries are available from `GET /api/goals` (docs/FOOD.md).

### What to build

`just vm upkeep-report SINCE UNTIL [--home X Y Z] [--assignments FILE]`, which saves the window with the existing `audit-window` recipe and runs a new `scripts/vm/upkeep-report.py` on it. `--audit FILE` reads a local window instead, so it also works on a file a soak already saved. It writes `.checks/upkeep-<ts>/summary.json` and prints a short text result. Sections:

- **Coverage:** share of the window with Tas in the world and within 32 m of home, and the share of perception samples that saw each in-scope fire and cooking station. A soak where Tas was home for less than 80 % of the window, or a structure seen in less than 80 % of those samples, is `inconclusive`, not a pass.
- **Fires:** per fire ID, samples seen, samples lit, lit share at night (`time.night`) excluding the first 120 s after each dusk, the grace goal 5.1 allows, lit share while a cooking slot within 3 m is `cooking` or `done`, failing samples at fuel 0 under goal 5.1's rule (A fire at fuel 0 within the first 120 s after dusk passes, because the bot can only start stoking once it sees night and may have to fetch fuel from a chest; any night sample at fuel 0 after that, or any sample at fuel 0 with food cooking over it, fails. Fires may run dry by day.), the longest unlit stretch at night in seconds, and the count of matched fuel receipts (the add_to task from TKT-01M62VXZ4D (Plugin: add fuel and ore to machines and take their output) once it exists; count none before then).
- **Fuel reserve:** for each fuel type, stored fuel at every fresh chest snapshot plus the pack, against the reserve, and any sample below it (goal 5.1: stored fuel never drops below the reserve).
- **Cooking:** per station and slot, transitions into `burnt` from a sample that was `cooking` or `done`, and each `done` slot's time until it emptied.
- **Assignments:** given the assignment table (a JSON file, or read from the controller once TKT-01M62VXZ84 exists), the last snapshot of each chest seen during the window, the units of each assigned item found outside its chests, and the units of each assigned item in the pack above its `keep` (goal 5.3). Don't-touch chests are excluded. For each full-chest event (a deposit refused as full), the delay to the owners' chat line naming the item and count (goal 5.3 needs at most 10 s) and the gap since the previous such line for the same chest (at least 5 min). Snapshots older than the window start count as unknown, not as correct.
- **Pantry:** each household food target's count at every fresh inspection, scored from the audit's `container` snapshots of the food chests, with the targets and minimums taken from settings (or a `--targets FILE`). Episode outcomes, uncertain pending actions, come from a `--goals FILE` saved from `GET /api/goals` at the end of the window; that endpoint keeps only recent receipts, so it is not used for counts. Without the goals file the uncertain-episode check is inconclusive. It reports the counts at each inspection and the episodes that ended with an uncertain pending action. It also fails on any death in the window, from the audit's death events, because goal 5.6 requires 0 deaths.
- **Incidents:** deaths, plugin task failures with reason `internal`, storage-shortfall chat lines and perception scan traces with `truncated: true` (the evidence TKT-01M62VXZM9 needs).

Each section has its own verdict (`pass`, `fail`, `inconclusive`) against thresholds given on the command line, so one report serves goals 5.1, 5.2, 5.3 and 5.6. Base the parsing on `scripts/vm/audit-summary.py`. Keep full frames on disk and print a bounded summary, as the other trial tools do.

### Files

- `scripts/vm/audit-summary.py`: the audit-window reader to start from
- `scripts/vm/trial-status.py`: how compact summaries are written
- `scripts/vm/vm.just`: add `upkeep-report` and `test-upkeep-report` beside `audit-summary`
- `justfile`: add `vm::test-upkeep-report` to the `ci` recipe beside the other `vm::test-*` checks
- `docs/TESTING.md`: one section on reading the report

### Verification

`just check vm test-upkeep-report` runs the self-test against small fixture windows: a fire that goes out at night, a fire dry by day and relit within 120 s of dusk (pass), one still dry 120 s after dusk (fail), fuel dipping below the reserve (fail), an assigned item held in the pack above keep (fail), a full chest reported after 10 s or twice within 5 min (fail), a pantry target below its minimum after the first episode (fail), an uncertain episode (fail), a death (pantry fails), no goals file (pantry inconclusive), a slot that burns, a misplaced assigned item, a window with Tas away from home (inconclusive), and a clean pass. `just check ci` runs it. No game is needed.

## Acceptance criteria

- [ ] just vm upkeep-report writes .checks/upkeep-<ts>/summary.json with coverage, fires, fuel reserve, cooking, assignments, pantry and incidents sections, each with a pass, fail or inconclusive verdict
- [ ] A window with Tas within 32 m of home for less than 80 % of it, or a structure seen in less than 80 % of those samples, is inconclusive, never pass
- [ ] just vm test-upkeep-report runs in just ci against fixture windows: a fire out at night, a fire relit at dusk, a fire still dry after dusk, fuel below the reserve, pack above keep, a late or repeated full-chest line, a pantry shortfall, an uncertain episode, a death, a missing goals file, a burnt slot, a misplaced assigned item, Tas away from home, and a clean pass
- [ ] docs/TESTING.md says how to run the report and read each section
