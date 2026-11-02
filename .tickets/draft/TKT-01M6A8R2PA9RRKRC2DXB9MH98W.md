---
schema: 4
id: TKT-01M6A8R2PA9RRKRC2DXB9MH98W
title: Slice hostile guidance construction within the frame budget
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - movement
  - plugin
  - tooling
  - live
assignees: []
milestone: M4
parent: TKT-01M6864352N8E28YCGHA19AN90
origin: null
dependencies:
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-01T00:39:28Z
updated_at: 2026-11-02T16:30:42Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

The guided household return in PR #138 succeeded without injury, but its bounded visibility graph is built synchronously when ordinary search exhausts. In `.checks/audit-window-GWZnrB.jsonl` at2026-11-01T00:36:18Z, a16-point graph with602 terrain hints built in13.65ms; the worst measured local slice was15.01ms, above the2ms target. Subsequent guidance builds were0.07–0.23ms when the straight corridor became eligible. This is a retained performance limitation, not a failed return or permission to relax physical checks.

Investigate slicing graph construction/hint sampling across physics updates, or reusing terrain hints with explicit world/load/terrain invalidation. Preserve the ordinary-first bridge policy, at-most-one guided retry,4096-hint bound, same-task ownership, overall deadlines and fresh physical/hostile movement authority. Include construction in measured frame work; do not hide the spike by only timing later A* expansions. Compare the same captured fixture and a live return before selecting the implementation.

## Acceptance criteria

- [ ] Guidance construction yields within the existing frame target with bounded total work and diagnostic accounting.
- [ ] Ordinary-first bridge handling, initially-away detours and live physical authority remain covered.

## Notes

**agent:claude-code/renki-plan-m3** at 2026-11-02T15:58:15Z

### Milestone review: may already be done; not an M3 goal

M3 grooming, 2026-11-02: this serves none of M3's goals 3.1-3.5. Commit a5b1dc3 (Guide coastal detours around eight threats with sliced graph building), which landed about 90 minutes after this ticket was filed, made `ThreatGuidance` build through an enumerator with `BuildSteps` and `BuildStepLimit` (`plugin/Renki.Core/Movement/ThreatGuidance.cs`). Check whether hint sampling is also sliced. If both are, this ticket can be closed with that evidence. If not, it fits M4 (safe to leave running) as a frame-budget fix. Parent set to TKT-01M6864352, where the return work lives.

Files: `plugin/Renki.Core/Movement/ThreatGuidance.cs`, `plugin/Renki.Plugin/BodyTasks.cs` (`Guidance()`), `plugin/Renki.Tests/ThreatGuidanceTests.cs`. Verify: `just check plugin test-core ThreatGuidance` against the captured return fixture `plugin/Renki.Tests/Fixtures/hostile-return.json`; the live comparison needs the VM lease (TKT-01M6EF5DTG).

**agent:claude-code/renki-plan** at 2026-11-02T16:30:42Z

Applied on 2026-11-02 by the planning pass: milestone set to M4. No M3 goal needs it, as the earlier note says. It is a frame-budget fix, which suits M4 (safe to leave running). First check whether commit a5b1dc3 already sliced hint sampling as well as graph building; if it did, close this ticket with that evidence. The owner can unschedule it.
