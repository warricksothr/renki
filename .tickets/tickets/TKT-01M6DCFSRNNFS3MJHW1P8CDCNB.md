---
schema: 4
id: TKT-01M6DCFSRNNFS3MJHW1P8CDCNB
title: "OBS adapter: leaked connection per poll, unchecked source restore"
type: bug
status: ready
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
claim: null
archive: null
created_at: 2026-11-02T05:42:34Z
updated_at: 2026-11-02T06:10:52Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-review
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

## Acceptance criteria

- [x] status() reuses one client and a test proves open connections stay bounded over many polls
- [ ] The source-loss trial restores the window with its own context and fails if the restore fails
- [ ] configure repairs an existing capture source's settings

## Notes

**agent:claude-code/renki-review** at 2026-11-02T05:48:47Z

Done in the review-fixes PR: one package-level client with IdleConnTimeout 30 s and MaxIdleConnsPerHost 2. TestStatusPollsReuseOneConnection fails with 50 connections for 50 polls on the old code and passes now. The panel server now also sets IdleTimeout 2 min. Still open: the source-loss trial restore and configure repair.
