---
schema: 4
id: TKT-01M6701ZJS66ZY78FVG10TSWMC
title: Bound model-assisted procedure revisions and recovery
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - llm
  - safety
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M6701PBVRTHD0SGRE45F42Q5
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T18:09:52Z
updated_at: 2026-10-30T18:09:52Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Add the bounded model revision loop proposed in docs/PROCEDURES.md after deterministic recovery exhausts its alternatives. Supply only objective, allowed capabilities, relevant fresh observations, distinct failures and remaining budgets. A model proposes a draft revision or supported alternative; validate source/schema/capabilities and replay scenarios before authorized activation. Preserve the previous version for rollback.

Persist request/token/time budgets through restart and revision. Revisions inherit original scope, permissions, deadline, spent budgets, cancellation and uncertain intents. A model cannot fabricate completion evidence or override native refusals. Retain compact decisions and artifact references for context efficiency. Human input is for actual missing resources or authority.

Stable reusable scripts may later become reviewed Go compositions or C# routines, preserving evidence contracts and replay fixtures. Visual authoring can compile to this same representation later.

## Acceptance criteria

- [ ] Planner uses bounded relevant context and durable request/token/time budgets
- [ ] Proposals validate and replay before activation, with rollback and inherited authority and spent budgets
- [ ] Replay proves revisions cannot revive cancellation, repeat uncertain actions or fabricate success
