---
schema: 4
id: TKT-01M6DCFS3KT9T88SW31YECTR1B
title: "Plugin: a throwing task freezes the bot's last inputs"
type: bug
status: done
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - safety
assignees: []
milestone: M3
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:42:33Z
updated_at: 2026-11-02T06:07:37Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-review
  name: ""
extensions: {}
---

## Description

Nothing catches an exception thrown inside a running body task.

### Where

- `plugin/Renki.Core/Tasks/TaskRunner.cs` `FixedTick` calls `body.Tick(dt)` with no try/catch. It runs from the Harmony prefix on `PlayerController.FixedUpdate` (`plugin/Renki.Plugin/RenkiPlugin.cs:129`, `Driver.cs:62`).
- `RenkiPlugin.cs` `Handle(msg)` (around :264) calls task constructors. An exception there that is not `TaskRefused` aborts the frame and leaves the request with no reply.
- `RenkiPlugin.cs:159` and `:208`: when `Safety.RecoveryReturn` fails to parse, `recoveryReturnOrder` is null. `GuardRecovery` then calls `recoveryReturnOrder.FoodReserveSeconds(...)` before `RecoveryReturn.Decide` gets the chance to fail closed.

### Failure

- **Escaped exception:** any null or index error in about 9k lines of Unity-facing code escapes. `SetControls` is skipped from then on, and Valheim keeps the last movement, run and attack inputs. The bot walks or swings forever, the controller never hears a failure, and a task with `timeoutS: 0` (follow, escort) never times out.
- **Malformed RecoveryReturn:** a malformed setting throws on every frame. No messages are handled, no state is sent, and the recovery logout never runs.

### Fix

- In `FixedTick`, wrap `body.Tick` in try/catch. On an exception, `body.Cancel()`, then `host.Idle()`, then reply `failed` with reason `internal` and the exception message (bounded), and log once.
- In `Handle`, do the same around `StartBody` and reply `rejected`.
- Null-guard `recoveryReturnOrder`, or start an immediate `LogoutRecovery("invalid startup return order")`.
- Add Core tests: one with a body task that throws, and one with a null return order.

## Acceptance criteria

- [x] An exception from a body task's Tick cancels it, idles the body and reports failed with reason internal
- [x] An exception while starting a task replies rejected instead of no reply
- [x] A malformed Safety.RecoveryReturn no longer throws every frame; recovery fails closed
- [x] Renki.Core tests cover a throwing task and a null return order

## Implementation plan

Wrap body Tick, StartBody and RunInstant in TaskRunner with catch-all handlers: cancel, idle, and report failed/rejected with a new protocol reason 'internal' and a bounded detail. Null-guard the two uses of recoveryReturnOrder before RecoveryReturn.Decide, which already refuses a null order. Tests in CoreTests and RecoveryReturnTests.

## Notes

**agent:claude-code/renki-review** at 2026-11-02T05:48:46Z

Added protocol reason 'internal' in docs/BRIDGE-PROTOCOL.md 6.6, Go protocol.ReasonInternal and C# Wire.ReasonInternal. The controller has no reason allowlist, so older controllers just see an unfamiliar reason. Still unverified in the game: no live exception has been forced. The guard is exercised by Core tests only.

**agent:claude-code/renki-review** at 2026-11-02T06:07:37Z

Follow-up in the same PR: every Cancel and Idle now goes through Release, which clears the runner's state first and contains throws. TaskRunner takes an optional fault sink, which the plugin wires to the BepInEx log with full stack traces. Not compiled here: Renki.Plugin needs VALHEIM_DIR, see TKT-01M6DCFST1. The plugin-side edits are three lines in RenkiPlugin.cs.

## Summary

TaskRunner now contains exceptions from body ticks, task starts and instant tasks. It reports them with reason 'internal' and idles the body, and a malformed RecoveryReturn no longer throws every frame. Landed in the review-fixes PR.
