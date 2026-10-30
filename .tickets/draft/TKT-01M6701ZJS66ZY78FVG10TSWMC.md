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

Implement bounded model-assisted revision only after the deterministic host and agent tools in docs/PROCEDURE-TOOLS.md work. Interactive coding agents and the future automatic planner must use the same capability/diagnosis/evidence/draft/validate/replay/compare interfaces. Persist evidence-linked feedback with separate observed facts, inferred causes, remedies, authorship and uncertainty. The needs-plan fingerprint triggers one bounded improvement episode instead of a model call per tick.

Persist reserved and consumed model request/token/time budgets, proposed edits, jobs and version lineage across restarts and revisions. Unknown remote usage retains its conservative reservation until reconciled. Proposals inherit original goal scope, deadline, permissions, cancellation, uncertainty and spent native budgets. World text and diagnostics are untrusted data, not authority to widen capabilities. Native refusal or absent food is not fixable by invented policy evidence.

Measure verified success and all terminal categories with denominators/cohorts, unchanged-action retries, damage/authority/evidence violations, efficiency, context/tool/token cost and proposal rollback rate. Keep replay/simulation/live results separate. Candidate publication, live canary and stable promotion require their own scoped authority; no automatic promotion or model call enters a native physics tick. Promote reusable patterns to reviewed Go/C# routines only after explicit contract, corpus, performance and live-evidence criteria; local native policy research remains a separate draft experiment.

## Acceptance criteria

- [ ] One fingerprinted improvement episode uses compact relevant evidence and durable request/token/time/edit limits across revisions and restart, including uncertain remote usage
- [ ] Feedback distinguishes immutable observed facts from model hypotheses and records superseding corrections and exact version/scenario/evidence provenance
- [ ] Proposals validate and compare on pinned scenarios before authorized candidate publication/canary/stable promotion, with prior-version rollback
- [ ] Revisions cannot revive cancellation, reset native budgets, alter scope/permissions or replay an uncertain action; untrusted world text cannot expand authority
- [ ] Verified success metrics include denominators, unresolved/cancelled cohorts, expected negatives, violations and context cost, with simulated and live evidence distinct
- [ ] Failed or exhausted improvement stops with an actionable shortfall and cannot call a model per tick or bypass missing native capabilities
