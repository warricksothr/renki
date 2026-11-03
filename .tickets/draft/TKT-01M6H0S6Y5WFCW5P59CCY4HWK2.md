---
schema: 4
id: TKT-01M6H0S6Y5WFCW5P59CCY4HWK2
title: Keep live task probes from interrupting automatic retreat
type: bug
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - tooling
  - vm
  - safety
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-03T15:34:57Z
updated_at: 2026-11-03T15:34:57Z
created_by:
  id: agent:codex/home-survey-delivery
  name: ""
updated_by:
  id: agent:codex/home-survey-delivery
  name: ""
extensions: {}
---

## Description

A direct live task probe can replace automatic retreat between readiness observations. In the new-home trial at2026-11-03T15:28:11.609Z, just vm task-check dispatched chest withdrawal t-27 while automatic flee t-25 was already running. The new direct body order cancelled flee. Existing recovery resumed as t-28 after0.22s and cancelled the withdrawal. Both Greydwarf hits occurred before the interruption; do not attribute either to this probe race.

Evidence .checks/task-n9cjsrp1.json and .checks/audit-window-OzbhE9.jsonl. Tas survived at0.5HP and ultimately reached the new-home house through ordinary native movement. The probe should make an intentional replacement distinguishable from stale operator assumptions, with an atomic ownership/admission check so a preflight read alone cannot race active recovery. Preserve explicit owner takeover semantics, emergency stop, honest uncertain receipts, and the existing single body-task contract. Decide and document the default/override behavior before implementing; do not globally deny authorized manual movement based only on low health. Also keep the operator's separate open-gates mistake visible in the incident record.

## Acceptance criteria

- [ ] Atomic admission distinguishes a deliberate replacement from stale probe ownership, with documented override semantics.
- [ ] Regression covers recovery starting between preflight observation and probe dispatch, without cancelling recovery by default.
- [ ] Explicit owner takeover and emergency stop retain their existing contracts; live interruption evidence is retained honestly.
