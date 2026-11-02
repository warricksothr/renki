---
schema: 4
id: TKT-01M6DCFSRNNFS3MJHW1P8CDCNB
title: "OBS adapter: leaked connection per poll, unchecked source restore"
type: bug
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - vm
  - tooling
assignees: []
milestone: M4
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/renki-m4-obs
  branch: t3/obs-restore
  worktree: /home/sothr/.t3/worktrees/renki/t3-c765f67a
  commit: 26dd1f039e9942423eadf30ab229fec1aee78bfb
  session: null
  claimed_at: 2026-11-02T18:48:56Z
  expires_at: null
archive: null
created_at: 2026-11-02T05:42:34Z
updated_at: 2026-11-02T18:53:35Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-m4-obs
  name: ""
extensions: {}
---

## Description

Two defects in the Windows OBS capture adapter.

### 1. One leaked connection per poll (high)

`internal/obscapture/adapter.go:32-33,69`: `status()` builds a fresh `http.Client` with its own `http.Transport` on every call. The response is drained and closed, so its connection goes into that one-use transport's idle pool, which nothing ever closes. The controller's `http.Server` sets no `IdleTimeout` either.

`Run` polls once or twice a second, which means about 3,600–7,200 open connections an hour, plus goroutines on both sides. Windows' default ephemeral range is 16,384 ports, so the VM runs out of outbound sockets within hours, and the controller's LLM and bridge calls fail with it.

Fix: one package-level client with `IdleConnTimeout`, or `DisableKeepAlives: true`. Add a test that runs `status` many times against an `httptest.Server` and asserts the open connection count stays bounded, for example with `ConnState` counting.

### 2. The source-loss trial can pass with OBS still on a fake window (medium)

`internal/obscapture/obs.go:268-275` restores the capture with `defer set(Window)`. That throws away the error and reuses the trial's `ctx`, which is already cancelled on interrupt. `windows-action trial-source-loss` then reports success with the capture on "Renki nonexistent window…". After that, `VerifySource` fails every second and `configure` cannot repair it, because it only creates missing inputs. This contradicts docs/WINDOWS-VIDEO.md:119.

Fix:
- Restore with a fresh bounded `context.Background()` timeout.
- Return the restore error with `errors.Join`.
- Make `configure` reapply the settings of an existing source.

### Goal

Goal 4.6 (review closed). The leak fix in criterion 1 is what lets goal 4.2's 100-connection limit pass.

### Files

Line numbers refer to commit f1a5483; at 0b932ef the restore is `defer set(Window)` at `internal/obscapture/obs.go:228` and `Configure` is at :325.

- `internal/obscapture/obs.go`, `internal/obscapture/obs_test.go`
- `cmd/obs-adapter` (the `trial-source-loss` action)
- `docs/WINDOWS-VIDEO.md` (around :119)

### Verification

- `just check controller test-packages ./internal/obscapture ./cmd/obs-adapter`, with tests that a cancelled trial context still restores the window, that a failed restore fails the trial, and that `Configure` rewrites an existing source's settings.
- `just check ci`.
- Optional live confirmation under the VM lease: `just streaming windows-action trial-source-loss`, then `just streaming windows-status` shows the capture back on the Valheim window. The criteria do not need it.

## Acceptance criteria

- [x] status() reuses one client and a test proves open connections stay bounded over many polls
- [x] The source-loss trial restores the window with its own context and fails if the restore fails
- [x] configure repairs an existing capture source's settings

## Implementation plan

SourceLossTrial: OBS requests run on context.WithoutCancel(ctx) (each Call keeps its own 3 s bound), because coder/websocket closes the connection when a read's context is cancelled, which would make any restore impossible after an interrupt mid-request. Only the wait between polls watches ctx. The restore runs in a defer with a fresh context.Background() 5 s timeout, sets the exact window, then VerifySource, and joins its error onto the trial's with errors.Join. Configure: for an input that already exists, SetInputSettings with overlay true and the spec settings, so a source left on a wrong window (or wrong mode/priority) is repaired before the final VerifySource. Tests in obs_test.go against a stateful mock OBS: a trial cancelled mid-poll still restores and verifies the window; a refused restore fails a trial that otherwise passed; Configure rewrites an existing Source's settings. Update docs/WINDOWS-VIDEO.md around the trial paragraph.

## Notes

**agent:claude-code/renki-review** at 2026-11-02T05:48:47Z

Done in the review-fixes PR: one package-level client with IdleConnTimeout 30 s and MaxIdleConnsPerHost 2. TestStatusPollsReuseOneConnection fails with 50 connections for 50 polls on the old code and passes now. The panel server now also sets IdleTimeout 2 min. Still open: the source-loss trial restore and configure repair.

**agent:claude-code/renki-m4-obs** at 2026-11-02T18:53:35Z

Trial requests now run on context.WithoutCancel: coder/websocket closes the connection when a read's context is cancelled, so the old code lost the connection on an interrupt mid-request and no restore could have worked (the new cancel test fails on the old code with 'OBS response unavailable'). The restore uses a fresh 5 s context, then VerifySource, joined with errors.Join. Configure reapplies settings (overlay) to both existing inputs, the standby browser source as well as the game capture. Live confirmation under the VM lease was skipped by instruction; the VM was not touched and this is not deployed.
