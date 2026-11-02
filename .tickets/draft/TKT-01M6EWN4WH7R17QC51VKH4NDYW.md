---
schema: 4
id: TKT-01M6EWN4WH7R17QC51VKH4NDYW
title: "Behaviours: loot pauses and resumes an owner's follow"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - chat
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
created_at: 2026-11-02T19:44:21Z
updated_at: 2026-11-02T20:15:45Z
created_by:
  id: agent:claude-code/renki-m2-loot
  name: ""
updated_by:
  id: agent:claude-code/renki-wave1
  name: ""
extensions: {}
---

## Description

Follow-up to TKT-01M6EFHE2N (Behaviours: loot template picks up drops near the owner), which shipped loot for the idle-near-owner case and left follow pre-emption out.

### What is missing

While Tas follows an owner the body is never idle, so the engine never starts the built-in `Loot` instance: `session.TryAutomationFor` refuses a body with a running task, and an owner's follow is a direct order. Goal 2.2 needs pickups during a follow.

### What to build, once the owners agree (TKT-01M62VXYQB)

- Let the `loot` template's `preemptFollow` parameter (today rejected by validation) borrow the body from an owner's follow: stop the follow without counting as a new owner order (so `ControlRevision` does not clear queued work), run the pickups, then start the same follow again, as `leaveResume` in `internal/chat/brain.go` does after a death.
- Bound the detour so Tas cannot fall behind the follow's leash.
- A new owner order during the detour wins and the follow is not restarted.

### Files

- `internal/session/automation.go` (borrowing a follow), `internal/behaviour/engine.go`, `internal/behaviour/loot.go`, `internal/chat/brain.go`

## Acceptance criteria

- [ ] With preemptFollow on, loot stops a running owner follow, picks up, and starts the same follow again, in a test
- [ ] A new owner order during the detour wins and no follow is restarted, in a test

## Notes

**agent:claude-code/renki-wave1** at 2026-11-02T20:15:45Z

The adaptive loot reach (TKT-01M6EYEC1M, TKT-01M6EYEMRVAG415PX2A10XSXDC, TKT-01M6EYEMTXMJN0GSF2Y03CDB12) is most useful during a follow, when the anchor stops and Tas stands by. Without this ticket loot still only runs when Tas is idle, so promote this alongside them.
