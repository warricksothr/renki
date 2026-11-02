---
schema: 4
id: TKT-01M6EFHEG5TSEF99CS0TBYY0TF
title: "combat-trial: time an owner stop sent mid-melee"
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
dependencies:
  - TKT-01M6EFHEBCYEFSTZNJJPFGAJCP
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:55:08Z
updated_at: 2026-11-02T15:55:35Z
created_by:
  id: agent:claude-code/renki-plan-m3
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m3
  name: ""
extensions: {}
---

## Description

Goal 3.2 in docs/ROADMAP.md needs at least one `just vm combat-trial` run where an owner stop ends combat within 2 s. The runner (`scripts/vm/combat-trial.py`) submits one native kiting goal and waits for the outcome. It has no way to send a stop mid-fight or to time what follows.

A trial cannot type in-game chat as an owner. Chat `stop` (`cmdStop` in `internal/chat/brain.go`) and the panel's `POST /api/act/stop` (`PanelAct` in `internal/app/panel.go`) both dispatch the same `protocol.TaskStop`. So the trial can use the panel action, and the goal's wording should say "owner stop (chat or panel)".

### Outcome

- `--stop-after-contact SECONDS` (0 to 30): once `state.combat` reports melee for the trial's task, wait that long, then `POST /api/act/stop`.
- `summary.json` records `stopSentAt`, `combatEndedAt` (first state with no `combat`, or a combat for a different task ID), `stopLatencyS`, and `attacksAfterStop` (any `attack` task sent in the 10 s after the stop, from the audit window).
- The run passes the stop check when `stopLatencyS <= 2` and `attacksAfterStop == 0`. A stopped run is not counted as an observed Greyling death.
- Cleanup is unchanged, and still never stops a newer human order.

### Files

- `scripts/vm/combat-trial.py` and `scripts/vm/combat-trial-test.py`
- `scripts/vm/combat-evidence.py` if the after-stop audit scan fits there
- `docs/SELF_DEFENSE.md` ("Verification") and `docs/TESTING.md`

### Verification

- `just check vm test-combat-trial`, with fixtures for: stop latency under 2 s, over 2 s, and an attack sent after the stop.
- `just check ci`

Depends on the retreat fix that stops a remembered hit from restarting combat after a stop. Without it the live run can fail `attacksAfterStop`.

## Acceptance criteria

- [ ] combat-trial --stop-after-contact N sends one panel stop N seconds after melee starts for the trial's task
- [ ] summary.json records stopSentAt, combatEndedAt, stopLatencyS and attacksAfterStop
- [ ] vm::test-combat-trial covers a fast stop, a slow stop and an attack after the stop
- [ ] docs/SELF_DEFENSE.md Verification describes the option
