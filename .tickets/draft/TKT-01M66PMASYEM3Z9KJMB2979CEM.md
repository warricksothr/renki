---
schema: 4
id: TKT-01M66PMASYEM3Z9KJMB2979CEM
title: Compose persistent exploration goals with discovery and return proof
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - movement
  - live
  - vm
assignees: []
milestone: M7
parent: TKT-01M66PFAYA1NTW2JHNRDKBK42E
origin: null
dependencies:
  - TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
  - TKT-01M62N2557R401S3E6JECSAPN4
  - TKT-01M66PMAPGXNJHB49E7WMGEBYS
  - TKT-01M66PMAQSBK2TH076X7M5FGAM
  - TKT-01M62VXZ0J9Q2WH4XN4E6N7F27
  - TKT-01M6EFQP110HAMEANK85XT7F5J
  - TKT-01M6EFQP437YT124TQ25299BJV
  - TKT-01M6EFQP5XBXHS67Q03PXCZPT0
  - TKT-01M6EFQP851CG8DV29FSHZ7KMP
  - TKT-01M6EFQPB8FCXDRVQ5ZT1Q1JQ1
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T15:25:07Z
updated_at: 2026-11-02T15:59:32Z
created_by:
  id: agent:codex/renki-exploration
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
extensions: {}
---

## Description

Extend the existing persistent goal host with a typed exploration specification and implemented child runners per docs/EXPLORATION.md after M2/M3. Persist scope, baseline coverage, selected/failed frontiers, original budgets and return state. The host, not a policy/model, validates useful new observations and supported arrival at the return anchor. Partial/needs-plan outcomes remain distinct from success. This ticket remains draft.

### Split (rollout step 4, plus the step 5 benchmark)
Umbrella for five one-PR children:
- TKT-01M6EFQP11 (Explore goal kind: spec, validation and durable budgets)
- TKT-01M6EFQP43 (Explore goal runner: legs, discovery and return proof)
- TKT-01M6EFQP5X (Replay scenarios for exploration goals): goal 7.5
- TKT-01M6EFQP85 (VM: expedition-trial supervisor with coverage and return evidence): the missing measurement recipe, offline
- TKT-01M6EFQPB8 (Earn three consecutive bounded expedition passes on the VM): goals 7.3 and 7.4, live

### Files
- internal/goal/store.go, internal/goal/runtime.go, internal/goal/policy.go, internal/goal/return.go, internal/app/goals.go, internal/inspection/inspection.go, scripts/vm/vm.just, scripts/vm/combat-trial.py

### Verification
`just check controller test-packages ./internal/goal ./internal/app`, `just check vm test-expedition-trial`, `just check ci`, then `just vm expedition-trial` (VM lease) with three consecutive passing summary.json files. Done when the five children are done.

## Acceptance criteria

- [ ] Host enforces durable total time/travel/attempt/wait/area/biome limits and reserved return allowance across all runners
- [ ] Completion requires novel observed coverage plus fresh supported arrival evidence; repeated timestamps, movement success and shared map reveal alone cannot complete it
- [ ] Replay covers uncertain restart, missing telemetry, checkpoint failure, blocked return, owner stop/takeover and preserved budgets; a bounded land benchmark records actual discovery/return or an honest partial outcome
