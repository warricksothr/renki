---
schema: 4
id: TKT-01M6DDFTDDA6D23TTDJPAA2SV0
title: "VM: haul report from the audit for owner-haul sessions"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - vm
  - tooling
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T06:00:03Z
updated_at: 2026-11-02T15:57:27Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m2
  name: ""
extensions: {}
---

## Description

M2's first scene, where the owner chops and the bot hauls, needs a human playing, so no supervised trial can stage it. docs/ROADMAP.md goal 2.2 measures it afterwards from the audit log instead.

### What to build

A `just vm haul-report` recipe, with an offline `test-haul-report` in `just ci` like the other trial checks. It reads an audit window (start and end, or the last N minutes) and reports:

- **Dropped:** items that appeared within 30 m of an owner (perception drops and owner-attributed events), by prefab. Drops that the loot template's deliberate-drop rule ignores (TKT-01M6EFHE2N, a pure function: within 2 m of an owner with no tree, log or stump breaking within 10 m in the last 10 s) are counted separately as **Deliberate** and left out of the ratio. Reuse that function, or a port tested against the same cases, so the two cannot drift.
- **Picked up:** of those drops, what the bot picked up, from pickup receipts.
- **Pickup ratio:** per prefab and in total.
- **Trips:** home trips, with each deposit's matched receipt counts, the trigger (fill fraction or the pile being gone, with the idle seconds) and the time back at the site.
- **Incidents:** deaths, drowning-risk refusals and failed deliveries, with reasons.

Write the result to `.checks/haul-<ts>/summary.json`, plus a short text summary. Base it on `scripts/vm/audit-summary.py` and the existing trial-status helpers.

The report reads audit files only, so building it needs no VM lease. Running it on a real owner session is TKT-01M62N12HJ's evidence (its criterion 5). That live criterion moved there from this ticket so this one stays offline.

### Why

Without a number, "the bot keeps hauling" stays a judgement call. Goal 2.2 needs a pickup ratio of at least 80 % and at least 3 trips in each of 2 owner sessions.

### Files

- `scripts/vm/audit-summary.py`, `scripts/vm/trial-status.py`, `scripts/vm/trial-status-test.py` (helpers and test style)
- `scripts/vm/vm.just` (beside `audit-summary`), root `justfile` (`ci` list)
- `docs/AUDIT.md` (the record shapes), `docs/TESTING.md`

### Verification

- `just check vm test-haul-report` against a fixture audit with known drops, pickups, one deliberate owner drop (excluded from the ratio) and two trips, then `just check ci`.
- On any retained audit window: `just vm haul-report SINCE UNTIL` writes summary.json with the fields above.

## Acceptance criteria

- [ ] just vm haul-report summarises drops near owners, pickups, trips, deposits and incidents for an audit window
- [ ] An offline test-haul-report runs in just ci against a fixture audit
- [ ] The summary.json fields (dropped, picked up, ratio, trips with trigger, incidents) are documented in docs/TESTING.md
