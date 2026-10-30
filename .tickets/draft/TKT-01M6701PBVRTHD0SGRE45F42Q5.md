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
updated_at: 2026-10-30T18:09:42Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Implement the versioned authoring contract proposed in docs/PROCEDURES.md over the existing restricted Starlark evaluator. Packages pin immutable source/hash, objective and state schemas, allowed capabilities, required permissions, host success predicates and replay scenarios. Active goals pin an exact version, including subprocedure dependencies.

Provide typed capability discovery, draft, validate, dry-run/replay, publish, diff and trace operations suitable for users and model authors. Separate authoring, publishing and operating permissions. Preserve exact grammar examples, compact node/phase diagnostics and proof references; retain full traces as artifacts. Keep runtime AST restrictions. Broader untrusted syntax needs an OS-limited evaluator worker first.

## Acceptance criteria

- [ ] Immutable packages and pinned versions validate against host capabilities and evidence predicates
- [ ] User/model draft, validation, replay and publication interfaces preserve operation permissions
- [ ] Deterministic scenario bundles and compact decision traces cover success and adverse recovery
