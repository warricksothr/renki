---
schema: 4
id: TKT-01M6701B73BXT58GXFS0FYNZK8
title: Generalize durable goals for multistep food search
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - safety
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T18:09:31Z
updated_at: 2026-10-30T18:09:31Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Follow docs/PROCEDURES.md, researched at the owner's request. Extend the existing restricted Starlark goal host with typed capabilities, checkpointable child phases, per-native-task intents/budgets/receipts and a persistent host-owned candidate/attempt ledger. The existing runtime extracts one combat attack and discards the child; adding template names alone is insufficient.

Use food resupply as the first generic procedure. Fresh inventory and matched transfer receipts prove progress; remembered chest contents only rank candidates. An inspected empty chest advances search. Count repetition by action, target and relevant observation fingerprint, separately from global budgets. Preserve owner cancellation, scope, permissions and uncertain-action rules. Do not introduce another language or broaden the interpreter.

Existing bounded test-supervisor resupply is a reference implementation, not evidence that this production goal is already available. Combat and M2/M3 acceptance remain separate.

## Acceptance criteria

- [ ] Generic child phases checkpoint and account for every native task; existing combat behavior is preserved
- [ ] Empty or stale chest hints advance a durable candidate ledger without repeating unchanged actions
- [ ] Food success requires matched receipts and fresh scoped inventory evidence
- [ ] Replay covers restart uncertainty, partial transfers, owner takeover and persistent budgets; bounded live resupply passes
