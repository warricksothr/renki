---
schema: 4
id: TKT-01M6EFH8QFHNPK6AHWKMGART94
title: "Agent: remind_me schedules a later wake"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - llm
  - controller
assignees: []
milestone: M6
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies:
  - TKT-01M62VXZBNPYYFFFVT7QCSVTMS
  - TKT-01M62VXZAV8CYT5K4DYAQEZC2Q
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:55:02Z
updated_at: 2026-11-02T15:55:02Z
created_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
extensions: {}
---

## Description

Split from TKT-01M62VXZCV (Agent: autonomy levels, owner pins, ask_owner and reminders) so that ticket stays one pull request.

`remind_me(after, about)` is a Social tool the agent uses to schedule its own later wake: "check the wood chest again in an hour", "ask Sothr about the fire at dusk". When it comes due, the wake bus (TKT-01M62VXZBN, Agent: wake the LLM on world events, within a budget) receives a `reminder_due` event carrying `about`, and the turn runs under the normal wake budget.

### What to build

- The tool, with `after` bounded (for example 1 min to 24 h) and `about` bounded in length.
- At most N pending reminders per bot (default 5); the oldest is refused, not silently dropped, and the refusal goes back to the model.
- Reminders persist in the controller's `dataDir` and survive a restart; a reminder that came due while the controller was down fires once on start, not once per missed interval.
- Owners can list and clear reminders from chat or the panel.
- Each reminder set, fired, cleared or refused is logged.

### Files

- `internal/voice/voice.go`: the registry from TKT-01M62VXZAV
- the wake component from TKT-01M62VXZBN
- `internal/settings/instances.go` or `internal/goal/store.go`: atomic persistence patterns to follow

### Verification

- `just check controller test-packages` for the voice and wake packages, with fake time: a reminder fires once when due; the cap refuses the extra one; a restart keeps pending reminders and fires an overdue one once; an owner's clear removes it.
- `just check ci`.

## Acceptance criteria

- [ ] remind_me schedules a reminder_due wake carrying its about text, bounded in delay and count
- [ ] Pending reminders survive a restart, and an overdue one fires once
- [ ] Owners can list and clear reminders
