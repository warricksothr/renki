---
schema: 4
id: TKT-01M6EWN4WH7R17QC51VKH4NDYW
title: "Behaviours: passive mode lends an owner's follow to passive instances"
type: task
status: done
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
updated_at: 2026-11-03T04:18:17Z
created_by:
  id: agent:claude-code/renki-m2-loot
  name: ""
updated_by:
  id: agent:claude-code/passive-deploy
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

- [x] Instances have an idle or passive mode, replacing loot's preemptFollow; the built-in Loot is passive and on
- [x] While the followed player is settled, the engine lends the follow to the first ready passive instance and restarts the same follow after, in a test
- [x] A walking leader's follow is never lent, and leaving the settle area ends the run and restarts the follow, in a test
- [x] An owner order during a passive run wins and no follow is restarted, in a test
- [x] docs/BEHAVIOURS.md describes idle and passive modes and the passive list; just check ci passes
- [ ] Live, under the VM lease: Tas picks up felling drops while following a settled owner and falls back in behind

## Implementation plan

As built. This replaces the first plan.

### Session (internal/session)
- `Session.OwnerFollow()` reports the running owner follow: a non-final `follow` sent outside automation (or given back), the state's current task, and the only body task in flight.
- `TryLoanFrom(hello, world, followID)` reserves the body for exactly that follow. The handle's first body task marks the follow `Lent()`, but only while the follow is still running, and clears the mark if the send fails.
- `Automation.Return(ctx)` sends the same follow again (args, timeout, Origin) through the handle, only when it ended superseded or is still running. It records owner provenance and spends the handle after a successful send. It does not bump ControlRevision. When nothing was replaced it sends nothing, returns nil, and leaves the handle valid.

### Engine (internal/behaviour)
- `Definition.Mode` is idle (the default) or passive. Passive needs a `Passive` template (loot). `preemptFollow` is removed.
- `Passive.Settled` comes from loot's tracker for the instance's area shape. `View.Following` and the anchor are the followed player.
- While following, only passive instances whose player is settled queue or run, and the engine borrows with TryLoanFrom. On each tick of a loan it checks the settle area, and a reset cancels the run with "followed player moved on".
- `cancel` and `finish` call `giveBack` first. It sends the follow back, or logs `follow not replaced`, `follow not returned` or `could not return the follow`; the last one also notifies the player. `cancel` then stops the step if the handle is still valid. A stop error is swallowed only when the loan was really spent.
- `Status.Passive` and the audit attributes show the borrowed follow. InstallLoot upgrades an unedited old built-in.

### Brain
- There is no special case. A lent follow ends cancelled/superseded, which the brain already answers with silence, and the anchor is untouched. A lent follow that failed on its own is handled as a real end, including the death resume.

## Notes

**agent:claude-code/renki-wave1** at 2026-11-02T20:15:45Z

The adaptive loot reach (TKT-01M6EYEC1M, TKT-01M6EYEMRVAG415PX2A10XSXDC, TKT-01M6EYEMTXMJN0GSF2Y03CDB12) is most useful during a follow, when the anchor stops and Tas stands by. Without this ticket loot still only runs when Tas is idle, so promote this alongside them.

**agent:claude-code/renki-passive** at 2026-11-02T23:21:16Z

Design notes, as built (they differ a little from the brief):

- **preemptFollow** is removed, not deprecated. Validation always refused `true`, so no saved instance can hold it. Because loot decodes with DisallowUnknownFields, a request that still names it now fails as an unknown field.
- **Settle tracking stays in Loot.** Passive templates implement `Passive.Settled(params, player, now) (since, ok)`, and loot answers from the tracker for that instance's area shape. The engine compares `since` on every tick of a loan, so an area reset ends the run.
- **The restart is sent by the session**, through `Automation.Return`, with the original task's args, timeout and Origin. It never goes through the brain, so `Brain.Anchor` and the `origin.follow` that rides on the task are untouched. The returned task is recorded with owner provenance, so it can be lent again.
- **A loan that sent no step** (the run found nothing) leaves the original follow running and sends nothing. Without that, a second follow would replace the first for nothing.
- **Retreat**: flee and defend go through Session.Dispatch, so they revoke the loan like any owner order, and the follow is not restarted. That matches what retreat does to a plain follow today. Restarting a follow after retreat would be its own change.
- **Escort is never lent**, only `follow`.
- **Panel**: the panel has no general instance card yet; only the idle-food line reads `/api/behaviours`. So `Status.passive {follow, player}` on the API is the card's data, and the audit has `behaviour started` with `passive`, `follow` and `player`, and then `follow returned`, `follow kept` or `follow not returned`. The panel UI is left to whoever builds instance cards.

**agent:claude-code/renki-passive** at 2026-11-02T23:21:16Z

Live check (criterion 6), for the owner, under the VM lease. Not run: this job did not touch the VM.

1. `just vm lease <you>`, then deploy this branch's controller (docs/VM.md). The plugin is unchanged.
2. Check that Loot is passive: `just vm panel` GET `/api/behaviours`, where `Loot` should show `"mode":"passive"`. An old saved Loot that is still the unedited built-in is upgraded on start, and the log says `built-in behaviour made passive`.
3. In game, say "tas, follow me", walk about 20 m and stop beside a beech. Tas must not pick anything up while you walk.
4. Fell the beech and stand within about 6 m of the spot for at least 5 s. Expected: Tas leaves its follow distance, picks up the Wood, Resin and BeechSeeds within reach (8 m at first, growing toward 30 m), and then falls in behind you again. It says nothing about the follow ending.
5. Start another pickup run, then walk off more than about 8 m. Tas should stop at once and follow.
6. During a pickup, say "tas, stop". Tas must stop and not resume following.
7. Check the audit: `just vm audit -last 10m` should show `behaviour started … passive=true follow=t-N player=<you>`, then `follow returned` (steps 4 and 5) and `follow not returned reason="new order"` (step 6).
8. `just vm release`.

**agent:claude-code/renki-passive** at 2026-11-02T23:21:26Z

Corrects two commands in the previous note. The deploy in step 1 is `just vm deploy-controller`. The audit in step 7 is `just vm audit 1 -last 10m`.

**agent:claude-code/renki-passive** at 2026-11-03T00:08:08Z

This corrects the design note: the audit record `follow kept` is now called `follow not replaced`. The records are `follow returned`, `follow not replaced`, `follow not returned` and `could not return the follow`. The last one also tells the followed player through Notify.

**agent:claude-code/renki-lead** at 2026-11-03T00:32:21Z

Supersedes the death trade-off in the earlier notes and the PR's first description. A death during a passive run now hands the lent follow to the chat brain. The session marks the loan `died` when a death revokes the handle, `Automation.DiedOnLoan` returns the replaced follow, the engine's giveBack passes it to `Options.ResumeFollow`, and that calls `Brain.ResumeLentFollow` → `leaveResume`. After the respawn Tas resumes the follow, as after a death during a plain follow. Covered by a session test (after a step, and not before any step) and an engine test. TKT-01M6FCYSS5 is archived as done here.

**agent:claude-code/passive-deploy** at 2026-11-03T04:18:17Z

Live session 2026-11-02, build a892cfa (#171 and #173), Nimah playing. Acceptance criterion 6 is still unticked.

- Passive Loot ran while Tas followed a settled Nimah: runs of 4, 2, 16 and 3 pickups, each ending with "follow returned" and Tas falling back in. Felling drops (Wood, Resin, BeechSeeds) were taken.
- Other runs were cut off 0.25–0.75 s after starting with "condition changed", over and over (20:25:28–20:26:07). Tas took one item, then stood by Nimah and only slowly bumped into the rest. Cause: a perception scan sent after the latest state frame is stamped ahead of it by up to 200 ms, and behaviours refused perception newer than state. So the `when.owner` condition and `freshPerception` failed in that window. Fixed in #174 (b5cc019, deployed 2026-11-03 04:16 UTC), which allows perception up to 1 s ahead of state.
- Not yet run: the owner's own stack left alone, walking off mid-pickup, an owner order during a pickup. Rerun the whole check on b5cc019, then tick criterion 6.
- Also seen: a portal broke the follow (`target_lost`, then Tas walked home). Filed as TKT-01M6FT1ZVW.

## Summary

Done in the PR from branch t3/passive-mode. Instances have an idle or passive mode. The built-in Loot is passive and on, and an unedited old built-in is upgraded. While an owner's follow runs and the followed player is settled, the engine lends the follow to the first ready passive instance (Session.TryLoanFrom). When the run ends it sends the same follow again (Automation.Return) without counting as an owner order. The run ends, and the follow restarts, when the player leaves their settle area. An owner order revokes the loan, and nothing restarts. Retreat counts as such an order. Tests cover lending and return, a walking leader, leaving the area, an owner order, an idle instance, the brain's silence and anchor, and the failure paths. docs/BEHAVIOURS.md has a new "Idle and passive" section. Criterion 6, the live check, is not done: the steps are in the notes. The panel has no general instance card; the data is `passive` on /api/behaviours.
