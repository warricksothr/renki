---
schema: 4
id: TKT-01M6ESNXKFSGRRVTD7C6EZ4P3P
title: "Flaky gate: app speech memory test times out at 2s under load"
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - tooling
assignees: []
milestone: M4
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T18:52:20Z
updated_at: 2026-11-02T20:15:45Z
created_by:
  id: agent:claude-code/renki-m3-marker
  name: ""
updated_by:
  id: agent:claude-code/renki-wave1
  name: ""
extensions: {}
---

## Description

`TestSpeechMemoryAndDeterministicLocationQuestionsIncludingPanel` in `internal/app/players_test.go:63` failed `just ci` on 2026-11-02 with "nothing from the controller for 2s". Four agents were running `just ci` on the same machine at the time. The package passed on its own straight after (`just check controller test-packages ./internal/app`, 7 s). The branch under test (t3/marker-health-baseline) changes only `internal/behaviour`.

A fixed 2 s wait for a controller reply under `-race` and a loaded machine makes the gate flaky. Find what the test waits on and make it wait on the event itself, or give it a bound that a loaded `-race` run meets.

## Notes

**agent:claude-code/renki-wave1** at 2026-11-02T20:15:45Z

A second internal/app test failed the same way: TestPlayerProfilesAuthorizeCommandsAndRouteMarcoWithoutVoice failed one full just ci run on the loot-template branch (TKT-01M6EFHE2N) and passed three times alone under -race and in every later full run. Look at the shared harness in internal/app tests, not just the one test.
