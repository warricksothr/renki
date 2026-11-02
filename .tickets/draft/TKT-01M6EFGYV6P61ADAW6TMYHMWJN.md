---
schema: 4
id: TKT-01M6EFGYV6P61ADAW6TMYHMWJN
title: "VM: run the 24 h soak and link its summary from the M4 epic"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - vm
  - live
  - safety
assignees: []
milestone: M4
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies:
  - TKT-01M6DDFTF6N15TE3ANQ5KWT1Q7
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
  - TKT-01M6DCFSHHG0DZJW21HTZGZGH4
  - TKT-01M6DCFSNTBFQYW1M4ZYEERSWC
  - TKT-01M6DCFS8Y4W0BW6KE3HYMZBCP
  - TKT-01M6DDBX5TPR0VD0RQ2CT7NY3M
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:54:52Z
updated_at: 2026-11-02T15:54:52Z
created_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

Goal 4.2's evidence: one 24 h run of `just vm soak` that passes, linked from the M4 epic. TKT-01M6DDFTF6 (VM: 24 h soak with crash, leak, growth and store checks) builds the supervisor; this ticket runs it for real once the fixes it would otherwise trip on have landed.

### What to do

- Take the VM lease (TKT-01M6EF5DTGGAB15GFMRMDT8KTA) for the whole window. The soak needs the game client, so it runs after the day's M3 trials (docs/ROADMAP.md, M3 and M4 overlap).
- Before the 24 h window, run goal 4.3's live half: force-kill the controller 3 times while it is saving (landmark batches rewrite memory.json every 30 s), restart it, and check that memory.json, goals.json and settings load with the home intact. These kills are outside the soak window, because the soak fails on any controller restart.
- Run `just vm soak 24` with the bot idle-working at home.
- Link `.checks/soak-<ts>/summary.json` and its verdict in a note here and on TKT-01M6DCF2AB.

### Files

- none expected; if the run fails, file the failure as a bug under TKT-01M6DCF2AB and link it here.

### Verification

- The soak's `summary.json` verdict is pass for 24 h, with no controller restart, at most 100 controller or obs-adapter TCP connections, projected audit growth under 2 GB a day, no store write error and no plugin `internal` failure.
- A note records the 3 kills and that every store loaded afterwards.

## Acceptance criteria

- [ ] Three forced controller kills during saves leave memory.json, goals.json and settings readable with the home intact, recorded in a note
- [ ] A 24 h just vm soak passes, and its summary.json is linked here and on TKT-01M6DCF2AB
