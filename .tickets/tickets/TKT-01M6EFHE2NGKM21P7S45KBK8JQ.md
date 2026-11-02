---
schema: 4
id: TKT-01M6EFHE2NGKM21P7S45KBK8JQ
title: "Behaviours: loot template picks up drops near the owner"
type: task
status: review
status_reason: null
priority: high
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
  actor: agent:claude-code/renki-m2-loot
  branch: t3/loot-template
  worktree: /home/sothr/.t3/worktrees/renki/t3-a7446e67
  commit: e0ea1b0833b2d4f26205e097313156e24c5bf0e5
  session: null
  claimed_at: 2026-11-02T19:21:12Z
  expires_at: null
archive: null
created_at: 2026-11-02T15:55:08Z
updated_at: 2026-11-02T19:44:31Z
created_by:
  id: agent:claude-code/renki-plan-m2
  name: ""
updated_by:
  id: agent:claude-code/renki-m2-loot
  name: ""
extensions: {}
---

## Description

Part of TKT-01M62N12HJ (Controller: loot and deliver-when-full behaviours for hauling), split out so each piece fits one pull request. Goal 2.2 in docs/ROADMAP.md needs the bot to pick up at least 80 % of the wood, resin and seeds an owner drops within 30 m of them. Today nothing picks up drops unless an owner configures a bounded `collect` instance with a fixed site (`internal/behaviour/collect.go` says it "does not take over follow, infer work sites ... or install default background loot jobs").

### What to build

- A `loot` template in `internal/behaviour` (a new `loot.go`, registered beside `collect` in the template map in `engine.go` `New`). Parameters: the items (default Wood, Resin, BeechSeeds, and the other felling products `scripts/vm/gather_trial.py` lists), a radius around the followed owner (default 30 m), and a weight fraction to stop at.
- It picks the nearest eligible drop from fresh perception, sends `pickup`, and checks the receipt against the next inventory frame, as `collect` does. Reuse collect's yield, capacity and receipt helpers rather than copying them.
- It leaves an owner's deliberate drops alone: a drop that appears within 2 m of an owner with no tree, log or stump breaking within 10 m in the last 10 s is ignored for 5 minutes. Put the rule in one pure function with its own tests.
- While following: the engine only starts an instance when the body is idle (`View.Idle`), and a follow task never finishes. Loot may pause the owner's follow, pick up, and start the same follow again afterwards, as `leaveResume` in `internal/chat/brain.go` does after a death. This needs the owner's agreement (see the note on TKT-01M62VXYQB). Until then, ship loot for the idle-near-owner case and keep the follow pre-emption behind a parameter that defaults off.
- One built-in instance, on by default as the behaviours epic decided on 2026-10-28, created on first start and editable like any other.

### Files

- `internal/behaviour/collect.go`, `internal/behaviour/engine.go` (template map, `triggers`, `drop_appeared`), `internal/behaviour/definition.go`
- `internal/behaviour/hauling_test.go` for the test style
- `internal/chat/brain.go` (`leaveResume`) for resuming a follow
- `docs/BEHAVIOURS.md`: document the template and the deliberate-drop rule

### Verification

- `just check controller test-packages ./internal/behaviour ./internal/chat`
- `just check ci`
- Live proof belongs to TKT-01M62N12HJ, through the haul report (TKT-01M6DDFTDD).

## Acceptance criteria

- [x] A loot template picks up eligible drops within its radius of the followed owner, verified against inventory receipts, in a test
- [x] A drop that appears at an owner's feet with no felling or splitting nearby is left alone for 5 minutes, in a test
- [x] A built-in loot instance exists and is on by default, and an owner can switch it off
- [ ] If follow pre-emption is enabled, the same follow is started again after the pickup, in a test
- [x] docs/BEHAVIOURS.md describes the template and the deliberate-drop rule

## Implementation plan

1. `internal/behaviour/loot.go`: a `Loot` template (pointer, shared by every instance and clone) with params `items` (default Wood, Resin, BeechSeeds, Feathers), `radius` (default 30 m, at most 40), `maxWeightFraction` (default 0.8, 0.1–0.9), `maxTargets` (default 16, 1–64) and `preemptFollow` (default false).
2. The runner embeds a carry-only `collectRun` and reuses its `fits`, `take`, `productGain` and pickup-recovery rules. It takes the nearest eligible drop within `radius` of a live owner in fresh perception, sends `pickup`, and requires a newer inventory frame with a measured gain before the next target.
3. The deliberate-drop rule is one exported pure function `DeliberateDrop(drop, at, owners, breaks)`, with its constants. The `Loot` template keeps a small observation tracker: it records tree/log/stump disappearances from fresh perception as breaks, and on a drop's first sighting asks the rule; a deliberate drop is ignored for 5 minutes.
4. Engine: register `loot` in the template map; call an optional `Observer` on templates each tick; an optional `Readiness` check gates trigger queuing so an idle timer does not queue empty runs. No change near `drop_appeared`.
5. Built-in instance `Loot`: timer trigger every 2 s, repeat `trigger`, `when.owner.radius` 40, priority -5, enabled; `Engine.InstallLoot` creates it when absent, called from `app.Serve` (not `New`, so the app tests that count rows stay valid). Disable is the off switch.
6. Follow pre-emption: the engine cannot borrow a body from an active follow (TryAutomationFor refuses busy bodies), and doing so needs the owner's agreement (TKT-01M62VXYQB). `preemptFollow:true` is rejected by validation for now; AC 4 stays unticked with a note and a follow-up.
7. Tests in `loot_test.go`; docs in docs/BEHAVIOURS.md.

## Notes

**agent:claude-code/renki-m2-loot** at 2026-11-02T19:44:31Z

Implemented on branch t3/loot-template. `just check ci` passes and `just review` reports no findings on the final head.

Evidence for the ticked criteria (all offline, in internal/behaviour/loot_test.go):
- AC 1: TestLootPicksNearestDropsNearTheOwnerAndChecksReceipts, TestEngineLootPicksUpADropNearTheOwner (the built-in instance dispatches `pickup` through the engine).
- AC 2: TestLootLeavesADeliberateDropForFiveMinutes, TestDeliberateDropRule, TestLootTakesDropsFromFellingAtTheOwnersFeet, TestLootExcusesWoodSeenAFrameBeforeItsLogBreaks.
- AC 3: TestBuiltInLootIsInstalledOnceOnAndCanBeSwitchedOff (installed on, disable survives a restart and a second install).
- AC 5: docs/BEHAVIOURS.md "Loot near owners".

AC 4 is left unticked. The engine cannot start loot during a follow: `session.TryAutomationFor` refuses a body with a running task, and pausing an owner's follow needs the owners' agreement (TKT-01M62VXYQB). `preemptFollow` exists as a parameter and validation rejects `true`. Filed TKT-01M6EWN4WH (Behaviours: loot pauses and resumes an owner's follow) as a draft for that piece.

Two decisions the next person needs:
- The rule now has a 2 s lag: a break up to 2 s after a drop's first sighting also excuses it, because a split log's wood can be seen a frame before the log goes. The haul report (TKT-01M6DDFTDD) should call `behaviour.DeliberateDrop` rather than restating the numbers.
- The built-in instance is installed from `app.Serve`, not `app.New`; deleting it brings it back on the next start, so disabling is the off switch.

Live check, still to run (needs the VM; not done here): with the controller deployed, stand idle near Tas, fell and split a beech within 30 m, and drop a stack of wood at your feet. Expect Tas to pick up the felling wood and leave the dropped stack for 5 minutes. Read the `behaviour started`/`behaviour step` lines for instance `Loot` in the controller log.

## Summary

Loot template, deliberate-drop rule (DeliberateDrop) and an on-by-default built-in Loot instance landed on t3/loot-template, for the idle-near-owner case. Follow pre-emption is reserved and moved to TKT-01M6EWN4WH; AC 4 stays open for it.
