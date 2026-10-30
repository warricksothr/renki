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

Follow docs/PROCEDURES.md and docs/PROCEDURE-TOOLS.md. Preserve M2/M3 priority and separate their acceptance from procedure infrastructure. Generalize the existing combat-specific goal host through explicitly typed reconstructible phases, one admitted food adapter at a time. There is no generic child checkpoint protocol today; registering arbitrary behaviour.Runner values is insufficient. Keep the restricted Starlark grammar and existing permission/body ownership boundaries.

Implement a host-owned durable candidate/attempt ledger, structured expected-negative versus execution-fault outcomes, and per-native-task intents, budget charging, run-scoped identities and receipts. A wrong remembered food hint must lead to a fresh scoped observation and another allowed chest. Repeat detection uses action, target and meaningful observation revision; distinct empty boxes are not identical retries. Food success requires the declared host predicate and matched transfer plus fresh inventory/consumption evidence appropriate to that predicate.

Add compact decision/evidence diagnostics and bounded scenario capture before automatic model integration. Reconcile interrupted actions only when retained proof suffices; otherwise preserve uncertainty and existing explicit-resume rules. Existing resupply_trial.py is reference/test supervision, not an implemented production goal. Unsupported crafting, repair, exploration and arbitrary durable behavior composition remain outside this slice.

## Acceptance criteria

- [ ] Explicit food adapter phases persist state and charge every native task without claiming arbitrary child-runner checkpointability; bundled combat behavior is preserved
- [ ] Wrong or empty food hints update a durable scoped candidate ledger and select another allowed chest; unchanged retries stay bounded across restart
- [ ] Host food completion requires declared receipt and fresh inventory/consumption proof; uncertainty, cancellation, current permissions and spent budgets cannot be bypassed
- [ ] Compact diagnosis links decisions, host context, scoped native receipts and candidate outcomes and distinguishes missing evidence from zero observed events
- [ ] Replay covers partial transfers, distinct empty chests, exhausted search, uncertain restart, checkpoint failure and owner takeover; separately authorized bounded live resupply earns its native evidence
