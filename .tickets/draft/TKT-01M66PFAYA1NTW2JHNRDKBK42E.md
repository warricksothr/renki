---
schema: 4
id: TKT-01M66PFAYA1NTW2JHNRDKBK42E
title: Exploration goals after M2 and M3
type: epic
status: draft
status_reason: null
priority: normal
due_on: "2027-01-17"
labels:
  - controller
  - movement
  - docs
  - live
assignees: []
milestone: M7
parent: null
origin: null
dependencies:
  - TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
  - TKT-01M62N2557R401S3E6JECSAPN4
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: children
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T15:22:24Z
updated_at: 2026-11-02T17:13:21Z
created_by:
  id: agent:codex/renki-exploration
  name: ""
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

Capture the owner's requested post-M2/M3 direction: persistent exploration goals supported by bounded native/controller routines for preparation, frontier selection, travel/survey, recovery and return. This is future design/backlog work, not permission to start autonomous exploration before the two milestones are complete.

Reuse the existing persistent goal/checkpoint machinery, per-world terrain/landmark/danger/player memory, waypoints, player permissions and panel. Extend the host with an exploration-specific specification, budgets and completion evidence; the current combat-only goal implementation does not already support exploration. Retain the existing explored-map coverage ticket rather than duplicating it.

See docs/EXPLORATION.md for the proposed first benchmark, evidence/provenance model, supporting behaviors, rollout order and open design decisions. Broader biome searches, resource prospecting, roads, portals, boats and overseas expeditions follow demonstrated bounded land exploration.

### M7 scope
Rollout steps 1 to 5 of docs/EXPLORATION.md. Step 6 (requested resources/biomes, planned resupply, larger expeditions) and boats, portals and camp building are outside M7.

### Children in rollout order
1. Coverage: TKT-01M62VXZ0J9Q2WH4XN4E6N7F27 (Know which parts of the map the bot has explored), split into measured coverage, visited ground and minimap knowledge
2. Frontiers: TKT-01M66PMAPGXNJHB49E7WMGEBYS (Select exploration frontiers and remember failed approaches)
3. Routines: TKT-01M66PMAQSBK2TH076X7M5FGAM (Add bounded expedition preparation, survey and return routines)
4. Goal and benchmark: TKT-01M66PMASYEM3Z9KJMB2979CEM (Compose persistent exploration goals with discovery and return proof)
5. Panel and chat: TKT-01M66PMAV9P2Z8QMR9N90T0TH1 (Show and control exploration progress in the panel)

### Measurement
"Measured terrain coverage" is canonical 4 m atlas samples (16 m2 each) first received during the expedition, inside the authorized circle, from this bot's source. It is loaded heightmap sampled within at most 64 m of the bot, not proof every object was seen. The evidence recipe is `just vm expedition-trial` (TKT-01M6EFQP85, VM: expedition-trial supervisor with coverage and return evidence), which writes `.checks/expedition-trial-<ts>/summary.json`. The exit goals are this epic's definition of done.

### Verification
`just check ci` for every offline child; live work takes the VM lease (TKT-01M6EF5DTGGAB15GFMRMDT8KTA, VM: exclusive lease for deploys and live trials).

## Acceptance criteria

- [ ] M2 and M3 are complete before exploration implementation is promoted
- [ ] Bounded land exploration earns new observed coverage and returns to its designated anchor with explicit completion evidence
- [ ] Preparation, travel/survey, retry selection, recovery and return respect native safety, permissions and durable goal budgets
- [ ] Panel shows planned/observed/visited coverage, discoveries, uncertain routes, failures and remaining expedition budget
- [ ] Replay and bounded live benchmarks cover meaningful discovery, safe return, blocked frontiers, interruption and honest partial outcomes

## Definition of done

- [ ] 7.1 Coverage kinds stay apart: offline tests show the measured-coverage query counts 0 m2 for re-confirmed, resent, minimap-only and visited-only ground, and visited ground keeps its y on raised floors (TKT-01M6EFQN74, TKT-01M6EFQN91, TKT-01M6EFQNBJ)
- [ ] 7.2 Frontier choice is honest in replay: at least 8 recorded scenes each return ranked candidates or 'no safe frontier', and 0 identical retries happen without new evidence (just check controller test-packages ./internal/explore; TKT-01M66PMAPGXNJHB49E7WMGEBYS)
- [ ] 7.5 Replay matrix: all 15 scenarios in docs/EXPLORATION.md 'Rollout and evidence' are named tests ending in a different allowed choice, partial or needs_plan, with budgets never reset (just check controller test-packages ./internal/goal; TKT-01M6EFQP5X)
- [ ] 7.8 M7 tickets tell the truth: every M7 child is done, or has a note saying what is left and is moved; both spikes are resolved or moved (git ticket list --milestone M7)
- [ ] 7.6 Panel shows the expedition: fixture tests render boundary, anchor, coverage, trail, frontier and budget with distinct planned/travelled lines (TKT-01M6EFQPDB), and a map snapshot from one 7.3 run is linked (TKT-01M6EFQPB8)
- [ ] 7.3 First bounded expedition: just vm expedition-trial reports goal73 true for 3 runs in a row (combat off; >=2,000 m2 new measured terrain inside a circle of radius <=150 m; a succeeded return plus a later state within 3 m / 1.75 m of the pinned anchor; elapsed <=10 min, work <=5 min, travel <=600 m; no death), with the three summary.json paths linked (TKT-01M6EFQPB8, TKT-01M6EFQP85)
- [ ] 7.4 Stop and partial are honest: a run with --stop-after 60, an owner stop through the panel, is idle within 2 s with the goal cancelled and tasksAfterStop 0; a no-frontier scene ends partial with no outbound leg (summary.json plus test name; TKT-01M6EFQPB8, TKT-01M6EFQP43)
