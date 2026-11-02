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

Goal 3.3 in docs/ROADMAP.md: 2 of 2 escort or guard runs engage a confirmed attacker of a protected player within 10 m, and stop on "stay out of it" with no further attack. `just vm escort-check` (`scripts/vm/escort-check.py`) only records health, the set of escort modes it saw and whether the escort ended. It does not say whether a defence happened, against what, how far from the owner, or how fast a stop took effect. So a run cannot be scored against 3.3.

### Outcome

`escort-check` writes, per defence episode (from `state.escort` mode `defend` and `state.combat` under the escort task ID):

- the creature's entity and prefab, what confirmed it, and the distance from the protected player when defence started. Per docs/ESCORT.md, "Defense and escape", a defence starts only on an observed AI target identifying a protected player, an AI target on Tas himself, or a recent hit against him. A **confirmed attacker of a protected player** is the first kind: record it as `trigger: protected_target` with that player, and the others as `self_target` or `self_hit`. If the plugin does not report which kind it was, record `trigger: unknown`;
- start, end and how it ended (target death observed, target lost, stopped, leader lost);
- `escortStopLatencyS`: when the escort task ends after an owner's "stay out of it" or "stop" chat line, the time from the line to the end of combat, and whether any `attack` or escort-defend started in the 10 s after. Read the chat line and tasks from the audit window for the run.

`summary.json` gains `defences` (a list) and `stop`, and a field `goal33` that is true only when at least one defence with `trigger: protected_target` started within 10 m of that player, an owner said "stay out of it" during or after that defence, the running defence ended (escort mode left `defend` and no `state.combat` remained under the escort task) before the run ended, and no `attack` or escort-defend started between that line and the end of the run. A run where "stay out of it" was never said has `goal33` false, because it cannot show the stop half of the goal. A `self_target`, `self_hit` or `unknown` defence never counts. The stop latency is reported for information; goal 3.3 sets no time limit on it (the 2 s limit is goal 3.2's).

The owner says "stay out of it" in game. The script does not send it.

### Files

- `scripts/vm/escort-check.py` and `scripts/vm/escort-check-test.py`
- `docs/ESCORT.md` ("Diagnostics and validation")
- Read `plugin/Renki.Plugin/EscortTask.cs` and `plugin/Renki.Core/Tasks/Escort.cs` for what `state.escort` carries; today `state.escort.reason` in defend mode is always "confirmed attacker targets a protected player", even when the threat was confirmed only by targeting Tas or by a recent hit on him (`Threats`, `Defend`), so do not read it as `protected_target`. Read the `trigger` field that TKT-01M6EHP58NR3825QRA6PP0ZHD1 (Escort: report whether a defence was for a charge or for Tas himself) adds; until it lands, record every defence as `unknown`. Do not add a protocol change here.

### Verification

- `just check vm test-escort-check` with fixtures for: a protected-target defence within 10 m, one beyond, a self-hit defence within 10 m (goal33 false), an unknown trigger (goal33 false), a defence with no stop said (goal33 false), a defence that continues past the stop to the end of the run (goal33 false), a stop with no restart, a stop followed by a restart, and no encounter.
- `just check ci`

## Acceptance criteria

- [ ] escort-check summary.json lists each defence with creature, start distance from the nearest protected player, start, end and how it ended
- [ ] summary.json records the latency from an owner's stay out of it or stop chat line to the end of combat, and any attack within 10 s after
- [ ] summary.json has goal33 true only under the rule in the description
- [ ] vm::test-escort-check covers the nine fixtures named under Verification
- [ ] docs/ESCORT.md describes the new fields
