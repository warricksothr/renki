---
schema: 4
id: TKT-01M630V80TFBMWT1B3XY4KP2QJ
title: "Spike: compare voice models on fixed chat scenarios"
type: spike
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - llm
  - terva
  - tooling
assignees: []
milestone: M6
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies:
  - TKT-01M630V7ZTEW2WR52HVWXR475M
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T05:06:42Z
updated_at: 2026-11-02T15:53:17Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
extensions: {}
---

## Description

On 2026-10-28 Nimah told the overloaded bot "Can you drop something and I
will pick it up". The local gemma took 26 s, answered "Aye, I am heavy as
a boulder. I will drop the wood now. I am quite heavy with this wood. I
will drop it now so we can keep moving." (repeating itself), and proposed
`stop`: it promised what no tool it had could do.

A replay harness with scenarios taken from the audit log (an order the
grammar misses, a request it has no tool for, small talk, a non-owner
asking for something) scores each model on latency, whether it chose the
right tool or none, and whether it promised only what it could do. That
decides the default model and whether a local model is good enough.

This is the first M6 ticket (docs/ROADMAP.md: choose the model on data). Its harness is reused afterwards: the query-tools ticket TKT-01M6EFCK6H (Voice: query tools for nearby, recall, behaviours and recent events) adds look-first scenarios to it, and the M6 goals cite its scores.

### What to build

- A fixed scenario set, checked in as data (for example `internal/voice/testdata/scenarios/*.json`): each holds a `chat.Request` (speaker, owner or not, text, summary, recent lines, offered tasks, gear) and the expected outcome: the tool that should be called with key arguments, or none, plus phrases that would be a false promise. At least 20 scenarios, at least 4 in each of: an order the grammar misses, a request with no tool for it, small talk, a non-owner asking for something, and an owner asking for something the bot is not allowed to do. Take the wording from real audit lines where they exist (for example the 2026-10-28 23:09 drop request).
- A small command that runs each scenario through `voice.Voice.Reply` against a named model and scores it: latency (ms), tokens in and out, tool chosen correctly (yes or no), honest (no promise without the matching tool call). It writes `.checks/voice-eval-<model>-<ts>/summary.json` with per-scenario results and totals (p50 and p95 latency, tool accuracy, honesty share, mean tokens per turn).
- A recipe for it, for example `just controller voice-eval MODEL [SCENARIOS]`. It reads the backend from `controller.local.json` or flags and the key from the environment variable the config names (`CPA_API_KEY` for cpa-api, docs/VM.md), and never prints the key.
- An offline test of the scorer with a fake runtime, so `just ci` covers the harness without network access.

### Comparison

Run every scenario 3 times against `claude-haiku-5-5` and `gemma-4-12B-heretic-v3-mtp` through cpa-api (`https://cpa-api.local.sothr.com/v1`), and any stronger local model the owner names. Record a table in a note: model, p50/p95 latency, tool accuracy, honesty, mean tokens per turn, and a recommendation. The default changes only when the owner agrees (owner decision).

### Files

- `internal/voice/voice.go` and `internal/voice/voice_test.go`: `Reply`, `prompt`, the fake runtime
- `internal/chat/brain.go`: `Request`, `Reply`, `Proposal`
- `cmd/controller/voice.go`: how the controller builds a voice from config
- `controller.just`: the recipe
- `docs/VM.md`: the cpa-api backend and key variable

### Verification

- `just check controller test-packages ./internal/voice` covers the scorer offline.
- `just check controller voice-eval <model>` against each model writes the summary.json named above. This needs network access to cpa-api and `CPA_API_KEY` in the environment, but not the game or the VM, so it is offline-lane work.
- `just check ci`.

## Acceptance criteria

- [ ] A just recipe replays a fixed set of chat scenarios against a chosen model and scores the replies
- [ ] haiku-5-5 and the local gemma are compared on latency, tool choice and honesty, and the result is recorded here
- [ ] At least 20 scenarios in five categories are checked in, and the scorer has an offline test in just ci
- [ ] Each run writes .checks/voice-eval-<model>-<ts>/summary.json with p50/p95 latency, tool accuracy, honesty share and mean tokens per turn
