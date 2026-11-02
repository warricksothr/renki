---
schema: 4
id: TKT-01M6EWN4WH7R17QC51VKH4NDYW
title: "Behaviours: passive mode lends an owner's follow to passive instances"
type: task
status: ready
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
updated_at: 2026-11-02T23:06:51Z
created_by:
  id: agent:claude-code/renki-m2-loot
  name: ""
updated_by:
  id: agent:claude-code/renki-lead
  name: ""
extensions: {}
---

## Description

Follow-up to TKT-01M6EFHE2N (Behaviours: loot template picks up drops near the owner), which shipped loot for the idle case only. The owner decided on 2026-11-02 (note on TKT-01M62VXYQB) that an owner's follow may be lent to **passive** instances, so that Tas works alongside a player who has stopped and falls back in behind them when they move on.

### What is missing

While Tas follows an owner the body is never idle, so the engine never starts an instance: `session.TryAutomationFor` refuses a body with a running task, and a follow is a direct order.

### What to build

- **Instance mode.** An instance is `idle` (the default, as today) or `passive`. This replaces loot's single `preemptFollow` parameter.
- **Lending the follow.** While an owner's follow is running and the followed player has been settled past the settle delay (the `Settle` tracker from TKT-01M6EYEMRV), the engine may lend the body to the first ready entry of the passive list:
  - it stops the follow without counting as a new owner order, so `ControlRevision` does not clear queued work;
  - it runs the instance, which works only inside the player's settle area and loot's reach around them;
  - it then starts the same follow again, for the same player and with the same origin, as `leaveResume` in `internal/chat/brain.go` does after a death.
- **Yielding.** The passive run ends and the follow restarts as soon as the followed player leaves their settle area. Any new owner order during the run wins, and no follow is restarted. Survival work (retreat, eating, recovery) still comes first.
- **Passive list.** An ordered list of passive instances, evaluated in order. The built-in list is the `Loot` instance, which becomes passive and stays on by default; work-alongside gathering joins the list in its own ticket. When Tas is idle, passive instances also run as they do today.
- **Visibility.** Instance cards and the audit show that a run was passive and which follow it borrowed from.

### Files

- `internal/session/automation.go` (lending a follow), `internal/behaviour/engine.go`, `definition.go`, `loot.go`
- `internal/chat/brain.go` (the follow it restarts, and the anchor kept across the loan)
- `docs/BEHAVIOURS.md`

### Verification

- Tests: a settled follow is lent and restarted; a walking leader's follow is never lent; leaving the area ends the run and restarts the follow; an owner order during the run wins with no restart; an idle-mode instance never borrows a follow.
- `just check ci`. Live, under the VM lease: follow, stop beside a felled tree, and watch Tas pick up the wood and fall back in behind.

## Acceptance criteria

- [ ] Instances have an idle or passive mode, replacing loot's preemptFollow; the built-in Loot is passive and on
- [ ] While the followed player is settled, the engine lends the follow to the first ready passive instance and restarts the same follow after, in a test
- [ ] A walking leader's follow is never lent, and leaving the settle area ends the run and restarts the follow, in a test
- [ ] An owner order during a passive run wins and no follow is restarted, in a test
- [ ] docs/BEHAVIOURS.md describes idle and passive modes and the passive list; just check ci passes
- [ ] Live, under the VM lease: Tas picks up felling drops while following a settled owner and falls back in behind

## Notes

**agent:claude-code/renki-wave1** at 2026-11-02T20:15:45Z

The adaptive loot reach (TKT-01M6EYEC1M, TKT-01M6EYEMRVAG415PX2A10XSXDC, TKT-01M6EYEMTXMJN0GSF2Y03CDB12) is most useful during a follow, when the anchor stops and Tas stands by. Without this ticket loot still only runs when Tas is idle, so promote this alongside them.
