---
schema: 4
id: TKT-01M6EFHEHM7X14XA1735WZCXMR
title: "escort-check: report defences, owner distance and stop latency"
type: task
status: draft
status_reason: null
priority: normal
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

Goal 3.3 in docs/ROADMAP.md: 2 of 2 escort or guard runs engage a creature that targets the owner within 10 m, and stop on "stay out of it". `just vm escort-check` (`scripts/vm/escort-check.py`) only records health, the set of escort modes it saw and whether the escort ended. It does not say whether a defence happened, against what, how far from the owner, or how fast a stop took effect. So a run cannot be scored against 3.3.

### Outcome

`escort-check` writes, per defence episode (from `state.escort` mode `defend` and `state.combat` under the escort task ID):

- the creature's entity and prefab, the protected player it targeted (when the plugin reports it) and the distance from that player when defence started;
- start, end and how it ended (target death observed, target lost, stopped, leader lost);
- `escortStopLatencyS`: when the escort task ends after an owner's "stay out of it" or "stop" chat line, the time from the line to the end of combat, and whether any `attack` or escort-defend started in the 10 s after. Read the chat line and tasks from the audit window for the run.

`summary.json` gains `defences` (a list) and `stop`, and a field `goal33` that is true only when at least one defence started within 10 m of a protected player and, if a stop was said, it ended combat within 2 s with no restart.

The owner says "stay out of it" in game. The script does not send it.

### Files

- `scripts/vm/escort-check.py` and `scripts/vm/escort-check-test.py`
- `docs/ESCORT.md` ("Diagnostics and validation")
- Read `plugin/Renki.Plugin/EscortTask.cs` and `plugin/Renki.Core/Tasks/Escort.cs` for what `state.escort` carries; if the targeted player is not reported, record it as unknown and do not add a protocol change here.

### Verification

- `just check vm test-escort-check` with fixtures for: a defence within 10 m, one beyond, a stop with fast end, a stop followed by a restart, and no encounter.
- `just check ci`

## Acceptance criteria

- [ ] escort-check summary.json lists each defence with creature, start distance from the nearest protected player, start, end and how it ended
- [ ] summary.json records the latency from an owner's stay out of it or stop chat line to the end of combat, and any attack within 10 s after
- [ ] summary.json has goal33 true only under the rule in the description
- [ ] vm::test-escort-check covers the five fixtures named in the description
- [ ] docs/ESCORT.md describes the new fields
