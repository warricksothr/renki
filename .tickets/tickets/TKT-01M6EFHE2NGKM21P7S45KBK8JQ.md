---
schema: 4
id: TKT-01M6EFHE2NGKM21P7S45KBK8JQ
title: "Behaviours: loot template picks up drops near the owner"
type: task
status: ready
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:55:08Z
updated_at: 2026-11-02T19:17:51Z
created_by:
  id: agent:claude-code/renki-plan-m2
  name: ""
updated_by:
  id: agent:claude-code/renki-wave1
  name: ""
extensions: {}
---

## Description

Part of TKT-01M62N12HJ (Controller: loot and deliver-when-full behaviours for hauling), split out so each piece fits one pull request. Goal 2.2 in docs/ROADMAP.md needs the bot to pick up at least 80 % of the wood, resin and seeds an owner drops within 30 m of them. Today nothing picks up drops unless an owner configures a bounded `collect` instance with a fixed site (`internal/behaviour/collect.go` says it "does not take over follow, infer work sites ... or install default background loot jobs").

### What to build

- A `loot` template in `internal/behaviour` (a new `loot.go`, registered beside `collect` in the template map in `engine.go` `New`). Parameters: the items (default Wood, Resin, BeechSeeds, and the other felling products `scripts/vm/gather_trial.py` lists), a radius around the followed owner (default 30 m), and a weight fraction to stop at.
- It picks the nearest eligible drop from fresh perception, sends `pickup`, and checks the receipt against the next inventory frame, as `collect` does. Reuse collect's yield, capacity and receipt helpers rather than copying them.
- It leaves an owner's deliberate drops alone: a drop that appears within 2 m of an owner with no tree, log or stump breaking within 10 m in the last 10 s is ignored for 5 minutes. Put the rule in one pure function with its own tests.
- While following: the engine only starts an instance when the body is idle (`View.Idle`), and a follow task never finishes. Loot may pause the owner's follow, pick up, and start the same follow again afterwards, as `leaveResume` in `internal/chat/brain.go` does after a death. This needs the owner's agreement (see the note on TKT-01M62VXYQB). Until then, ship loot for the idle-near-owner case and keep the follow pre-emption behind a parameter that defaults off.
- One built-in instance, on by default as the behaviours epic decided on 2026-10-28, created on first start and editable like any other.

### Files

- `internal/behaviour/collect.go`, `internal/behaviour/engine.go` (template map, `triggers`, `drop_appeared`), `internal/behaviour/definition.go`
- `internal/behaviour/hauling_test.go` for the test style
- `internal/chat/brain.go` (`leaveResume`) for resuming a follow
- `docs/BEHAVIOURS.md`: document the template and the deliberate-drop rule

### Verification

- `just check controller test-packages ./internal/behaviour ./internal/chat`
- `just check ci`
- Live proof belongs to TKT-01M62N12HJ, through the haul report (TKT-01M6DDFTDD).

## Acceptance criteria

- [ ] A loot template picks up eligible drops within its radius of the followed owner, verified against inventory receipts, in a test
- [ ] A drop that appears at an owner's feet with no felling or splitting nearby is left alone for 5 minutes, in a test
- [ ] A built-in loot instance exists and is on by default, and an owner can switch it off
- [ ] If follow pre-emption is enabled, the same follow is started again after the pickup, in a test
- [ ] docs/BEHAVIOURS.md describes the template and the deliberate-drop rule
