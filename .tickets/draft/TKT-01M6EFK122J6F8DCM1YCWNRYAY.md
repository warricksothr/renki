---
schema: 4
id: TKT-01M6EFK122J6F8DCM1YCWNRYAY
title: "Agent: live evidence for M6 goals: wake soak, worked example, owner"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - live
  - llm
  - vm
assignees: []
milestone: M6
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies:
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
  - TKT-01M6EFC7R886ACM2932G5ZESAP
  - TKT-01M62VXZBNPYYFFFVT7QCSVTMS
  - TKT-01M6EFG477P1D2WZZYFSAPR7ZY
  - TKT-01M62VXZCVVP0XXPDNYT5JYD51
  - TKT-01M62VXZE38GXGK324JHD0RQZD
  - TKT-01M62VXZF8N7JE0MS2J346V1R5
  - TKT-01M6EFCK6HFRF0Z7DNS8BR9GGZ
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:56:00Z
updated_at: 2026-11-02T15:56:00Z
created_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
extensions: {}
---

## Description

The M6 exit goals (docs/ROADMAP.md, M6) need evidence from the game, not only offline tests. This ticket collects it in one place so the implementation tickets stay offline and can run in parallel, and the single VM is used for M6 in one serialized block.

### Runs

1. **Wake soak.** Deploy (`just vm deploy`) with `agent.wake` on, autonomy ask, the chosen model, and the default wake budget. Leave the bot idle-working at home for 4 h. Then `just vm agent-report SINCE UNTIL` over the window. Counts: mean tokens per hour and the highest clock hour; wake latency p95 from event to turn start and to the first chat line; budget refusals; zero woken turns for danger events; zero actions dispatched without an owner's yes. If `just vm soak` (TKT-01M6DDFTF6, VM: 24 h soak with crash, leak, growth and store checks) has landed with a duration argument, run the 4 h under it so its crash and leak checks also apply.
2. **Worked example.** With a stock target set for wood above the remembered count (needs TKT-01M62VXZF8, Controller: stock targets and a stock-low event from chest memory, M6): `stock_low` wakes the agent, which looks (a query call before any action), asks the owner in chat, and on "yes" starts a gather instance with an `until` condition; the ending comes back as an event and the agent says so. Run once at ask and once at tell. Evidence: `just vm audit-window` for each run plus the report.
3. **Owner sessions.** At least 2 owner sessions totalling at least 2 h with autonomy ask, played normally. `just vm agent-report` over each counts proposals and the rejected share. This needs the owner to play; the agent schedules it, it cannot stage it.

Each run takes the VM lease (TKT-01M6EF5DTGGAB15GFMRMDT8KTA, VM: exclusive lease for deploys and live trials) first and releases it after restoring autonomy and waking to their prior values.

### Files

- `scripts/vm/vm.just`: `deploy`, `audit-window`, and the `agent-report` recipe from TKT-01M6EFC7R8 (Audit: agent report of tokens, wake latency and proposal outcomes)
- `docs/TESTING.md` and `docs/VM.md`: how live runs are recorded

### Verification

Live. Each run's `.checks/agent-report-<ts>/summary.json` path and audit window are linked in a note here and on the M6 epic, against the goal numbers in docs/ROADMAP.md. A run that misses a number is recorded as a miss with the reason, not rerun until it passes without saying so.

## Acceptance criteria

- [ ] A 4 h wake soak has an agent report linked here with tokens per hour, wake latency p95 and budget refusals
- [ ] The stock-low worked example ran once at ask and once at tell, with audit windows linked
- [ ] At least 2 owner sessions totalling at least 2 h have agent reports linked, with the rejected-proposal share
- [ ] Each result is compared with its M6 goal in a note on the epic, misses included
