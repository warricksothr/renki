---
schema: 4
id: TKT-01M6EFC7R886ACM2932G5ZESAP
title: "Audit: agent report of tokens, wake latency and proposal outcomes"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - tooling
  - llm
  - controller
assignees: []
milestone: M6
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:52:17Z
updated_at: 2026-11-02T15:52:17Z
created_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
extensions: {}
---

## Description

M6's exit goals bound tokens per hour, wake latency and the share of proposals an owner rejects. None of these can be read from the audit today without hand counting. This ticket adds the report that measures them, so the goals in docs/ROADMAP.md can cite a number.

What exists: every voice reply writes a `log` record `voice` (or `voice cost` when it failed) with `player`, `model`, `ms`, `tokensIn` and `tokensOut` (`internal/voice/voice.go`, around lines 145-185). Those records land in the audit log (docs/AUDIT.md, kind `log`). Nothing records why a turn started, when the triggering event happened, or what became of a proposal.

### What to build

- An offline report over audit files, as a Go command or a flag on `cmd/audit`, run by a new recipe `just controller agent-report [flags] [files]` (same file and time flags as `just controller audit`).
- A VM wrapper `just vm agent-report SINCE UNTIL` that fetches the window with the existing `audit-window` recipe and runs the report on it.
- It writes `.checks/agent-report-<ts>/summary.json` and a short text summary with:
  - **Tokens:** total, per hour (each clock hour and the mean), per model, and per trigger (`chat` or an event kind).
  - **Turns:** count per hour, failures, budget refusals (`ErrBudget`).
  - **Latency:** reply latency p50/p95/max from `ms`; wake latency p50/p95/max from an event's time to the turn's start and to the first chat line.
  - **Proposals:** proposed, accepted by an owner, rejected by an owner, lapsed, refused by the brain's checks, and the rejected share = rejected / (accepted + rejected). Lapsed are reported but not in the denominator.
  - **Tool calls:** per tool, with accepted and refused counts.
- Define the record names and fields the report reads in docs/AUDIT.md, so the wake ticket (TKT-01M62VXZBN, Agent: wake the LLM on world events, within a budget) and the autonomy ticket (TKT-01M62VXZCV, Agent: autonomy levels, owner pins, ask_owner and reminders) emit exactly those: for example `agent wake` (event kind, event time, turn start, groups offered), `agent tool` (tool, accepted or refused, reason) and `agent proposal` (id, outcome, who answered, when). Until they land, the report counts what exists and prints zero with "no records" for the rest, never a guessed number.

### Files

- `cmd/audit/main.go` and `internal/audit/timeline.go`: reading and filtering audit records
- `internal/voice/voice.go`: the existing `voice` and `voice cost` records
- `controller.just` and `scripts/vm/vm.just`: the two recipes (`audit-window` and `audit-summary` in vm.just show the pattern)
- `docs/AUDIT.md`: the record contract

### Verification

- `just check controller test-packages ./internal/audit ./cmd/audit` with a fixture audit holding voice, wake, tool and proposal records, including a failed turn, a budget refusal, a lapsed proposal and an hour with no turns.
- `just check ci`.
- Offline only. Running it over a real VM window is part of the M6 live evidence ticket, not this one.

## Acceptance criteria

- [ ] just controller agent-report prints tokens per hour, reply latency p50/p95 and per-model counts from the existing voice records in a fixture audit
- [ ] Wake latency, tool call outcomes and the rejected-proposal share are computed from the records docs/AUDIT.md defines, and missing records are reported as missing, not zero
- [ ] just vm agent-report SINCE UNTIL writes .checks/agent-report-<ts>/summary.json for a VM audit window
- [ ] The fixture test runs in just ci
