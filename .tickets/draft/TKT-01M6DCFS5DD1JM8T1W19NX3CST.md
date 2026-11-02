---
schema: 4
id: TKT-01M6DCFS5DD1JM8T1W19NX3CST
title: "Plugin: stale escort pointer, duplicate names, stuck chest requests"
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
assignees: []
milestone: M4
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:42:33Z
updated_at: 2026-11-02T15:56:49Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

Four small state bugs found in plugin review. Each is a one-line or few-line fix.

### 1. Escort's resupply exit leaves `EscortTask.Active` stale

`plugin/Renki.Plugin/EscortTask.cs:64-67` returns `Failed` without calling `Cancel()`. The runner only calls `Cancel` on tasks it ends itself, so `Active` and `MeleeTask.ClearAggressor()` are never cleared. `state.escort` keeps reporting the old ID with mode `wait`, and a stale attacker can linger for 15 s. Fix: `Cancel(); return Outcome.Failed(...)`. Then audit the other tasks that keep a static "currently running" pointer for the same pattern.

### 2. Two players with the same character name throw once a second

`plugin/Renki.Plugin/Reporter.cs:85-87` builds a dictionary keyed by `m_name` from `ZNet.GetPlayerList()`, and a duplicate key throws. That aborts the rest of `Update` (inventory, navigation, perception) every second, and `player_joined`/`player_left` stop entirely. Fix: key on the user ID, or group by name.

### 3. A chest whose open reply never arrives is refused forever

`plugin/Renki.Plugin/TransferTask.cs:83-84`: `Pending` clears only when a reply arrives. If the owning peer leaves first, every later deposit or withdraw at that chest fails with "still waiting for the previous chest request" until the chest unloads, which a home chest never does. Fix: record the request time and allow a new request after 30 s, still ignoring late replies to the old one.

### 4. Perception truncation is silent

`plugin/Renki.Plugin/Perceiver.cs:81`: when the collider buffer fills, resources become an arbitrary subset rather than the nearest ones, and only a trace every 30 s says so. Fix: add a `truncated` flag to the perception message (in the protocol doc, Go and C#), or sort by distance before trimming. Related: TKT-01M62VXZM (scans in dense bases can miss objects past 2048 colliders).

### Goal

Goal 4.6 (review closed). Item 2 also protects the 24 h soak (goal 4.2): a duplicate name aborts `Reporter.Update` every second.

### Files

Line numbers above refer to commit f1a5483 and still match at 0b932ef.

- `plugin/Renki.Plugin/EscortTask.cs` (the resupply exit around :64-67), `plugin/Renki.Plugin/KiteTask.cs`, `plugin/Renki.Plugin/TravelEncounter.cs`, `plugin/Renki.Core/Tasks/TaskRunner.cs` (`Release`, for the cleanup-after-throwing-Cancel note)
- `plugin/Renki.Plugin/Reporter.cs` (:85-87)
- `plugin/Renki.Plugin/TransferTask.cs` (`Pending` at :29, the check at :83-84, the reply patch around :203)
- `plugin/Renki.Plugin/Perceiver.cs` (:81), `docs/BRIDGE-PROTOCOL.md`, `internal/protocol/messages.go`, `plugin/Renki.Core/Protocol` (the perception message)

### Verification

- Move the decision logic for items 2 and 3 (player-list keying, the 30 s pending timeout) into Renki.Core where it can be tested, and cover it with `just check plugin test-core` plus the new test class name.
- Item 4: `just check controller test-packages ./internal/protocol` and `just check plugin test` (the fixtures round-trip in both languages).
- Renki.Plugin compiles: `just check plugin build` with `VALHEIM_DIR` set (it skips silently without it; see TKT-01M6EFGJZK).
- `just check ci`.

## Acceptance criteria

- [ ] EscortTask.Active is cleared on every exit path
- [ ] Two players with the same character name do not throw in Reporter
- [ ] A chest request with no reply for 30 s no longer blocks later transfers
- [ ] Perception says when the collider buffer truncated a scan

## Notes

**agent:claude-code/renki-review** at 2026-11-02T06:07:37Z

From the vuoro review of the review-fixes PR: TaskRunner.Release contains a throwing Cancel but then calls only host.Idle (Driver.Stand). KiteTask, EscortTask and TravelEncounter run their ClearCombat and Active cleanup after child cancels that could throw, so a partial Cancel can leave LookAt, the queued attack, dodge and block state, and the static Active pointers behind. Fix it here with the stale-pointer work: either do each task's cleanup in a finally block, or give ITaskHost a stronger reset that Release calls when Cancel throws.
