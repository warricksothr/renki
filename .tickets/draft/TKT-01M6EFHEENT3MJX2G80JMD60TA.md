---
schema: 4
id: TKT-01M6EFHEENT3MJX2G80JMD60TA
title: "VM: retreat-evidence report for goal 3.1 from an audit window"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - vm
  - tooling
  - safety
assignees: []
milestone: M3
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:55:08Z
updated_at: 2026-11-02T15:55:08Z
created_by:
  id: agent:claude-code/renki-plan-m3
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m3
  name: ""
extensions: {}
---

## Description

Goal 3.1 in docs/ROADMAP.md is measured from audit windows: HP falls below the retreat threshold with a threat within 15 m, `flee` is dispatched within 1.5 s, and the bot reaches its owner or home alive. Today someone reads those numbers by hand from `just vm audit-window` output. Retreats mostly happen during other trials (combat, gather, work trips), so the report must work on any audit window rather than only inside one supervisor.

### Outcome

A read-only recipe, for example `just vm retreat-evidence AUDIT`, that never contacts the VM. For each retreat episode in the audit it prints JSON with:

- the time HP first fell below `retreat.Low` (0.4 of max, `internal/retreat/retreat.go`) while a hostile creature was within `retreat.Near` (15 m), from `state` and `perception` records;
- the first `flee` task sent after that, and the latency;
- the flee's `toward` (owner, home or none) and its terminal status and reason;
- the distance to the owner or home at the end, using `retreat.There` (6 m) as "reached";
- whether a `died` event followed within 60 s;
- a pass/fail per episode against goal 3.1, and the episode's start position so a reader can tell whether it started on the raised base.

Episodes where a flee was dispatched by an injury rather than low HP are reported too, marked as such, because the retreat also flees on a confirmed hit at any health.

### Files

- New `scripts/vm/retreat-evidence.py` and `scripts/vm/retreat-evidence-test.py`. Follow `scripts/vm/recovery-evidence.py` and `scripts/vm/route-evidence.py` for shape.
- `scripts/vm/vm.just`: `retreat-evidence AUDIT` and `test-retreat-evidence`.
- `justfile`: add `vm::test-retreat-evidence` to `ci`.
- `docs/AUDIT.md` describes the record kinds; `docs/TESTING.md` should name the new recipe.

### Verification

- `just check vm test-retreat-evidence` with fixtures for: a pass toward an owner, a pass toward home, the 2026-10-31 raised-base failure (no movement, logout), a death, and a window with no episode.
- `just check ci`

## Acceptance criteria

- [ ] just vm retreat-evidence AUDIT prints one JSON episode per retreat with HP-cross time, flee dispatch latency, target, terminal status, end distance to owner or home, death, and pass or fail against goal 3.1
- [ ] It reads a retained audit file only and never contacts the VM
- [ ] vm::test-retreat-evidence runs in just ci with the five fixtures named in the description
- [ ] docs/TESTING.md names the recipe
