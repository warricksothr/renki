---
schema: 4
id: TKT-01M6EWN4WH7R17QC51VKH4NDYW
title: "Behaviours: passive mode lends an owner's follow to passive instances"
type: task
status: in-progress
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
claim:
  actor: agent:claude-code/renki-passive
  branch: t3/passive-mode-loot-follow
  worktree: /home/sothr/.t3/worktrees/renki/t3-cba95dcf
  commit: 104b36283b22a3609c5aace97ac6069d8cc7ae98
  session: null
  claimed_at: 2026-11-02T23:08:11Z
  expires_at: null
archive: null
created_at: 2026-11-02T19:44:21Z
updated_at: 2026-11-02T23:21:26Z
created_by:
  id: agent:claude-code/renki-m2-loot
  name: ""
updated_by:
  id: agent:claude-code/renki-passive
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

### Session (internal/session)
- `Session.OwnerFollow()` reports the running owner follow: a non-final `follow` the session sent outside automation, the only non-instant task in flight, and the state's current task. It returns its task ID and player.
- `TryLoanFrom(hello, world, followID)` reserves the body like `TryAutomationFor`, but accepts exactly that running follow. Its first dispatch marks the follow `Lent()`. The follow then ends `superseded`.
- `Automation.Return(ctx)` sends the same follow again, with the original args, timeout and Origin, through the still-valid handle. It records owner provenance on the new task and releases the handle. It does not bump ControlRevision. An owner order, death, reconnect or world change already invalidates the handle, so in those cases nothing returns.

### Engine (internal/behaviour)
- `Definition.Mode`: "" or "idle" (the default) or "passive". Passive needs a template that implements `Passive` (loot does) and a timer, event or manual trigger. Remove loot's `preemptFollow`; it was never saveable.
- `Passive.Settled(params, player, now) (since, ok)`: loot answers from its own settle tracker for that instance's area shape, so settle tracking stays in one place.
- `View.Following` is the owner follow (if the body runs one). While it is set, triggers queue only passive instances whose player is settled. `candidate` then picks only passive instances, and Tick borrows with `TryLoanFrom`. The engine keeps `e.loan` (follow task, player, area since). During the loan, `View.Anchor` is the followed player.
- Each tick of a loan checks the settle area. If it reset or the player was lost, the run is cancelled with "followed player moved on". `cancel` and `finish` give the follow back with `Return` while the handle is valid, instead of sending stop.
- `Status.Passive` holds the running loan's follow (task and player) for the card. The logs "behaviour started" and "follow returned" carry `passive`, `follow` and `player` for the audit.
- The built-in Loot becomes `mode: passive`. InstallLoot upgrades a saved Loot that is still the unedited old built-in default, and keeps its enabled flag.

### Brain
- `TaskDone` ignores a lent follow (`t.Lent()`): no speech, no home, no resume. The anchor is untouched, because the return is sent by the session, not through `ordered`.

### Panel and docs
- The instance card shows "Passive · borrowing <player>'s follow".
- docs/BEHAVIOURS.md: modes, the passive list, the yield rules, retreat.

### Tests
Engine tests with the fake plugin, session tests for loan and return, and a brain test.

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
