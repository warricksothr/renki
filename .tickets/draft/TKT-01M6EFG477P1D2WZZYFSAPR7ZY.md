---
schema: 4
id: TKT-01M6EFG477P1D2WZZYFSAPR7ZY
title: "Agent: wake budget with minimum gap, quiet hours and token caps"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - llm
  - controller
assignees: []
milestone: M6
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies:
  - TKT-01M62VXZBNPYYFFFVT7QCSVTMS
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:54:25Z
updated_at: 2026-11-02T15:54:25Z
created_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
extensions: {}
---

## Description

The budget half of TKT-01M62VXZBN (Agent: wake the LLM on world events, within a budget). M6's goals bound tokens per hour, so the wake bus needs limits that hold however many events the world produces.

Today the only limit is `Options.DailyTokens` in `internal/voice/voice.go` (`checkBudget`, `charge`, `rollDay`), a per-day cap shared by chat and everything else. On the VM it is 200,000 tokens (`llm.dailyTokens`, docs/VM.md). An owner's chat must never be starved by waking.

### What to build

- **Minimum gap** between woken turns (default 10 min), kept across merged events. An event inside the gap is held and merged into the next allowed turn rather than dropped; the held set is bounded.
- **Quiet hours** (local time, default none) during which nothing wakes, except events the owner marks urgent, if any.
- **Token share:** woken turns may spend at most a fraction of the daily cap (default 50 %), and at most an hourly cap (default 4,000 tokens per clock hour). Chat keeps the rest. A turn that would start over either cap is skipped and logged as a budget refusal.
- All of it in the settings store, so the owner can change it from the panel or chat and the agent cannot (agent writes to these keys are refused).
- Each skipped or held wake is logged so the agent report (TKT-01M6EFC7R8, Audit: agent report of tokens, wake latency and proposal outcomes) can count it.

### Files

- `internal/voice/voice.go`: `checkBudget`, `charge`, `rollDay`, `ErrBudget`
- `internal/config/config.go`: `LLM.DailyTokens`
- `internal/settings/settings.go`: keys, defaults, `ByAgent` refusal
- the wake component added by TKT-01M62VXZBN

### Verification

- `just check controller test-packages ./internal/voice ./internal/settings` plus the wake component's package, with fake time: 50 events in an hour produce at most 6 turns at a 10 min gap; no turns in quiet hours; the hourly cap stops turns and the next hour resumes; chat still replies when waking has used its share; the agent cannot change the budget keys.
- `just check ci`.

## Acceptance criteria

- [ ] Woken turns respect a minimum gap, with held events merged into the next allowed turn
- [ ] No turn wakes during quiet hours
- [ ] Woken turns stop at their share of the daily cap and at the hourly cap, while owner chat still gets replies
- [ ] The budget lives in the settings store, and agent writes to it are refused
- [ ] Every held or skipped wake is logged with its reason
