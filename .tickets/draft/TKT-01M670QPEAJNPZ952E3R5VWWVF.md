---
schema: 4
id: TKT-01M670QPEAJNPZ952E3R5VWWVF
title: Evaluate bounded native policies and choreography
type: spike
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - safety
  - tooling
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
  - TKT-01M62N2557R401S3E6JECSAPN4
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T18:21:43Z
updated_at: 2026-10-30T18:21:43Z
created_by:
  id: agent:codex/procedure-research
  name: ""
updated_by:
  id: agent:codex/procedure-research
  name: ""
extensions: {}
---

## Description

Research and implement only an explicitly approved experiment following docs/NATIVE-POLICIES.md. Compare parameterized native routines and a bounded declarative sequence/statechart with pinned MoonSharp/Jint candidates only when representative authoring needs justify them. Inventory the actual Valheim Unity/Mono/BepInEx runtime; netstandard2.1 is not sufficient evidence of deployed compatibility. Keep M2/M3 first and do not add another general runtime merely to express a short dance.

Prototype a pure Renki.Core finite sequence executor with typed snapshots/intents and a main-thread plugin facade, starting with stationary stand/look/emote. A moving shuffle needs an explicit guarded local-motion adapter. No exposed Driver, Unity/CLR objects, reflection, filesystem/network or arbitrary native callbacks. Preserve parent task ownership, existing native/session permissions, danger/owner/stop preemption and explicit cleanup. Do not persist interpreter stacks or resume interrupted short actions automatically.

Measure admission and per-tick time/allocation, bound states/actions/duration/travel, test every cancellation phase and delayed result epoch, and demonstrate why interpreter timeouts cannot preempt stuck host callbacks. Reuse compact agent diagnosis/version/replay tooling and report coverage gaps. Adoption requires two representative expression gaps, measured authoring benefit and passing deployment/resource/safety diagnostics; retaining declarative data without an interpreter is a valid result. Bounded live checks require separate authorization and evidence; no deployment or game changes are authorized by filing this draft.

## Acceptance criteria

- [ ] Document actual deployed compatibility and compare parameterized, declarative and pinned-interpreter options with explicit adoption/no-adoption evidence
- [ ] Pure finite executor and stationary example demonstrate bounded actions, typed facade and exact existing permission intersections without exposing raw controls
- [ ] Core corpus verifies deadlines, stale snapshots, refusal, cancellation at every phase, task epochs, neutral cleanup and session/world/death changes
- [ ] Benchmarks separate admission, steady ticks and telemetry allocations and expose cooperative timeout/slow-host-call limitations on the actual VM
- [ ] Agent tools diagnose phase/guard/native refusal and trace coverage; replay marks counterfactual physical outcomes unknown and versions are pinned per task
- [ ] Any separately authorized moving canary proves supported local travel and safe cleanup; short action interruption preserves Go host uncertainty and never auto-resumes
