---
schema: 4
id: TKT-01M6BZMS4KYXPA9PDQ22VAQ3GH
title: Expose shared behavior explanations and a live guard inspector
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - tooling
  - docs
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-milestones
  branch: t3/behaviour-inspection
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: f189ad3ff01a2504048aca72084a1140b2166a9c
  session: null
  claimed_at: 2026-11-01T16:57:57Z
  expires_at: null
archive: null
created_at: 2026-11-01T16:38:51Z
updated_at: 2026-11-01T17:23:27Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Implement the read-only operator inspection slice in docs/BEHAVIOUR-INSPECTION.md, using peeling and pantry goals as the first examples. Present one compact hierarchy from objective through procedure/behavior to the actual native body owner and phase. Report implementation support, authorization and current readiness separately, with guard values, freshness, stable reason codes, evidence links, interruption cause and remaining budgets. Do not infer missing native facts or invent rejected alternatives.

Use one structured explanation service for the panel and agent tools, aligned with docs/PROCEDURE-TOOLS.md. Add a read-only phase graph and guard inspector, then correlate decisions/intents/native outcomes/checkpoints into a bounded causal timeline. Link timestamped map and diagnostic camera evidence, preserving world/run/connection scope and explicit gaps; existing live video is not a synchronized replay archive. Planned paths/escape and actual observations must be visually distinguishable.

This slice precedes editable graphs and generic procedure publication. Native safety remains authoritative, viewing pause never pauses the bot, and no new script runtime is required. Keep agent summaries compact with bounded evidence retrieval rather than embedding full histories. Validate stale/offline state, unknown pursuit, changed body ownership, delayed second pursuers, blocked escape and pantry candidate-negative outcomes against actual runtime facts. Use the existing UI fixture/screenshot workflow for desktop/mobile presentation; recorded/simulated scenarios cannot certify live success.

## Acceptance criteria

- [x] Panel and agent diagnosis share scoped hierarchy, ownership, phase, reasons and budgets
- [x] Guard inspection separates support, permission, readiness, uncertainty and stale evidence
- [ ] Read-only graph and causal timeline link decisions, intents, outcomes and checkpoints with explicit gaps
- [ ] Map and camera evidence retain capture scope and distinguish planned from measured state
- [x] Peeling and pantry fixtures verify authoritative reasons, interruptions and offline behavior

## Implementation plan

Implement the first read-only inspection contract on a stacked branch above native peeling. Add an atomic session snapshot of state freshness, capabilities/effective controller permission, automation lease and outstanding task provenance; correlate the actual task origin with behavior/goal identity without serializing chat origins or trusting an engine status after lease revocation. Read-only projection must remain conservative across stale state, world/connection changes, native recovery and direct-order takeover before streamed task identity catches up.

Expose compact GET /api/inspection with lightweight goal choices and an optional selected goal. Project saved objective/phase/budgets, actual body ownership/native phase, typed guards with provenance and explicit unknown/stale states, a read-only possible-transition graph, and a bounded evidence ledger from persisted intents/outcomes/receipts. Explicitly disclose missing decision history rather than invent causal decisions, historical phases or simulated progress. Start with native peel and typed pantry food progress; all model/script publication remains future work. Use the same JSON for the panel and a just vm inspect recipe, bounded to a compact response.

Build an accessible Preact inspector with a goal selector, actual ownership summary, phase selector and guard table, evidence disclosure, explicit coverage gaps and task audit links where run/intent timestamps exist. Poll without changing body control and invalidate live presentation immediately on disconnect/scope change/request failure. Add deterministic projection and authenticated HTTP tests plus desktop/mobile fixture scenarios for pursuing/unknown members, quiet resets, food candidate-negative continuation, owner takeover, stale/offline state and untrusted strings. Keep unavailable map/camera capture links clearly unavailable in this first increment rather than claiming synchronized replay.

Run focused checks, UI examples and full just check ci; commit and just check review before push. Open/register a stacked PR, deploy controller-only while Tas is offline, and verify compact API/panel without a live retry into the last observed hostile crowd. Keep any full causal ledger or map/camera evidence AC honestly open if this increment cannot establish it.

## Notes

**agent:codex/renki-milestones** at 2026-11-01T17:20:10Z

### First read-only increment

Implemented a shared compact GET /api/inspection projection and just vm inspect, with a Preact Decisions inspector. It correlates the current exact automation lease, retained dispatch origin, behavior episode, current run and pending task. Direct dispatch takes over before streamed native state catches up; native recovery and offline/stale observations cannot inherit goal ownership. Combat and return intents now retain run/time scope; older unscoped evidence remains explicitly unknown. Native readiness, independent permission, competitor clearance and current escape validity are not inferred from necessary telemetry predicates.

Projection/authentication/session/goal checks pass through just check controller test-packages ./internal/inspection ./internal/session ./internal/app ./internal/goal. Panel unit checks cover late responses, selection/scope changes, failed refreshes, local expiry and GET-only requests. Desktop/mobile scenarios cover delayed pursuit, unknown guards, untrusted strings, owner takeover, stale state and offline pantry evidence. The inspector sits after Activity/Goals to preserve the existing monitoring layout; full layout scenarios verify it. Full gate and committed review results will be recorded in the PR.

### Remaining scope and field checks

AC 3 remains open: this increment provides read-only possible-transition descriptions and a bounded checkpoint evidence list, not a durable causal decision timeline. AC 4 remains open: map/video links are live views, not synchronized retained evidence. Those limitations are explicit in the UI/API/docs. Keep this ticket in progress for those next increments.

Tas remains offline at the previously observed unsafe position. No new live peel, kill, escape, return or pantry deposit is claimed from these fixtures. A native field trial still requires new evidence of a safe prepared start; do not replay the rejected login into the close hostile. Deployment for this increment is controller-only and must preserve the offline game state.

**agent:codex/renki-milestones** at 2026-11-01T17:23:27Z

### Deployment and transport verification

First increment committed as 8fcb2e6 and opened as PR 149 above PR 148. Full just check ci passed (.checks/20261101T171659Z-Yw6oTR.log), including all 58 desktop/mobile scenarios. Committed just check review f189ad3..HEAD passed without findings (.checks/20261101T172028Z-15SwcE.log). Reviewed the new desktop phase/ownership and mobile offline pantry screenshots; specific decisions are retained in .checks/panel-ui/visual-review.json.

Controller-only deployment completed with game not started. The authenticated browser and compact API report inspection-v1, offline, unavailable body ownership, 52 saved choices and no live native phase. Saved goals display without horizontal page overflow. The first SSH capture exposed existing Windows PowerShell Latin-1 response decoding and console-codepage output. Fixed panel-request.ps1 to read JSON explicitly as UTF-8 and emit UTF-8 stdout. The follow-up just vm inspect output decodes strictly as UTF-8 and round-trips the 0–3 s guard label (.checks/inspection-utf8-8fcb2e6.txt). No game login, task dispatch or live combat result is claimed. Causal-history and synchronized map/camera ACs remain open.
