---
schema: 4
id: TKT-01M67B7RKPCK73VAKHT9XQZ60Y
title: Research safe OBS broadcasting and offline profile tooling
type: spike
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - tooling
  - docs
  - vm
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/streaming-research
  branch: t3/research-twitch-streaming
  worktree: /home/sothr/.t3/worktrees/renki/t3-bf3847ce
  commit: 3ccd07d710e7c289f9bd46c8055f9c6ceaa7c8b3
  session: null
  claimed_at: 2026-10-30T21:25:50Z
  expires_at: null
archive: null
created_at: 2026-10-30T21:25:16Z
updated_at: 2026-10-30T21:32:00Z
created_by:
  id: agent:codex/streaming-research
  name: ""
updated_by:
  id: agent:codex/streaming-research
  name: ""
extensions: {}
---

## Description

The user explicitly requests isolated research and initial implementation of Twitch broadcasting for Renki from base 3ccd07d. Work only on t3/research-twitch-streaming in its existing separate worktree. Do not touch VM, deploy, restart, install OBS, change camera rates, stream, access credentials, or alter account settings. Do not depend on the separate panel layout branch.

Compare OBS/obs-websocket, FFmpeg and browser composition against current camera/map/panel code. Recommend real game capture with an optional dedicated read-only public-safe overlay. Implement only local secret-free configuration/planning tooling, validated with fixtures and no live connections.

## Acceptance criteria

- [ ] Compare capture options, privacy boundaries, recovery and setup prerequisites in durable design
- [ ] Provide secret-free offline profile validation and OBS planning with meaningful mock tests
- [ ] Pass just check ci and review committed changes; open a PR without deployment

## Implementation plan

The user's explicit scoped request authorizes this research/implementation ticket; promotion applies only to that scope, not unrelated drafts. Read base capture, panel authentication, VM desktop prerequisites and UI fixture automation before planning.

1. Verify OBS, obs-websocket v5, FFmpeg and Twitch primary documentation; distinguish supported APIs from unknown BLIGHT hardware/session behavior.
2. Write docs/STREAMING.md covering real video versus telemetry, capture/recovery/permissions/privacy, future read-only overlay and control contract, staged operator setup and unverified prerequisites.
3. Add standard-library-only offline profile validation and a bounded OBS diagnostic request/summary tool with synthetic fixtures; no network, secrets, control execution or UI branch changes. Route recurring tooling through root just recipes and CI.
4. Validate negative configuration and diagnostics cases and offline non-interference. Run just check ci before commits, just check review after committing, resolve findings and open/link PR against main. Leave live acceptance explicitly pending.

## Notes

**agent:codex/streaming-research** at 2026-10-30T21:31:25Z

Research recommends OBS in the game's interactive desktop, specific-window capture, static offline scene, optional separate loopback public-safe overlay and a narrow adapter around authenticated obs-websocket v5. Diagnostic JPEGs remain unchanged; 1 fps cannot provide smooth game streaming. GetSourceActive reports scene presence, not capture freshness.

Added a Python standard-library-only offline profile validator, inert five-request diagnostic batch, typed/redacted synthetic response summaries and mock tests through just broadcast-test / just broadcast-plan. No transport, OBS installer, UI, live account data or operational change. docs/STREAMING.md records exact sources and the current Twitch help-shell limitation; numerical profile limits are explicitly project policy.

Remaining later game checks: GPU/driver/encoder capability and headroom; Game/Window Capture across process/session/restart/minimization; reliable capture-health evidence; actual adapter/overlay privacy/permissions; owner-operated account setup; sustained upload/ingest/playback/audio checks. No live checks are authorized in this task.

**agent:codex/streaming-research** at 2026-10-30T21:32:00Z

Validation: just check broadcast-test passed initially; final just check ci passed in 104 seconds with 13 offline broadcast tests, required panel desktop/mobile browser scenarios, controller race/vet/build checks, 368 plugin tests and ticket validation. Full gate log: .checks/20261030T212949Z-1Tvn08.log. just broadcast-plan --response scripts/fixtures/broadcast/obs-response.json emitted only the expected typed synthetic summary, with captureHealth explicitly unverified. git diff --check passed. No new UI, so no new screenshots are claimed. Game plugin DLL compilation was skipped by the existing recipe because this worktree has no game assemblies.

The implementation rejects unknown CLI arguments without echoing values, unknown nested config fields, duplicate keys, non-finite input and oversized JSON. Diagnostic output excludes arbitrary OBS comments and payload strings. Next: commit review and PR handoff, then close this scoped research ticket; live integration remains later separately authorized work.
