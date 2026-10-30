---
schema: 4
id: TKT-01M6701PBVRTHD0SGRE45F42Q5
title: Add versioned procedure authoring and replay
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - docs
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M6701B73BXT58GXFS0FYNZK8
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T18:09:42Z
updated_at: 2026-10-30T18:21:43Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/procedure-research
  name: ""
extensions: {}
---

## Description

Implement the agent-usable offline tooling and release contracts in docs/PROCEDURE-TOOLS.md over the existing restricted Starlark evaluator. Begin with compact capability discovery, diagnosis, stable run-scoped evidence references and snapshot-paginated structured trace retrieval, including explicit truncation/retention gaps. Keep existing goal/audit APIs compatible. Add immutable packages and conditional draft replacement with exact grammar/schema diagnostics, canonical source/manifest/dependency digests and versioned checkpoint/predicate/capability contracts.

Expose capture, validate, replay and compare through one service shared by HTTP and future agent tools. Capture bounded pinned scenario inputs under quota before audit expiry. Recorded policy replay must report first counterfactual action divergence and unknown downstream world outcomes; host simulation is separately labeled and cannot create live proof. Provide a complete wrong-food-hint-to-other-chest diagnose/edit/replay workflow that detects a host candidate defect separately from policy defects.

Publication has two stages: validated/replayed immutable candidate artifact, then promotion of that exact version after a separately authorized bounded canary. Rollback changes future version selection and never undoes game effects or replays uncertain intents. Enforce strong revision preconditions, idempotent request submission and distinct read/author/publish/operate authority without expanding existing native permissions. Add agent-visible feedback, comparison metrics and source/behavior diffs. Automatic model authoring is separate follow-up work.

## Acceptance criteria

- [ ] Capability/diagnosis tools report implementation, permission and readiness separately with actionable stable codes, source locations and exact evidence links
- [ ] Bounded snapshot trace/evidence pagination exposes missing, expired and truncated classes, stable controller-run/task identity and redaction state
- [ ] Immutable package versions and conditional draft edits validate source, manifest, dependencies and host/checkpoint contracts without broadening Starlark syntax
- [ ] Scenario capture pins bounded inputs; recorded replay stops causal continuation at divergence and simulated receipts cannot satisfy live objectives
- [ ] Diagnose/edit/validate/replay/compare works offline for wrong food hints, empty alternatives and restart/permission cases with compact reproducible reports
- [ ] Validated candidates can publish before canary; stable promotion requires evidence for the exact candidate; authorized rollback preserves pinned active goals and world effects
- [ ] Publication/operation permissions, If-Match conflicts, idempotency-key conflicts and incompatible schema/version transitions have tested explicit outcomes
