---
schema: 4
id: TKT-01M62VXZM9R4S5X27NCXSMC4H0
title: "Perception: scans in dense bases can miss objects past 2048 colliders"
type: bug
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - plugin
assignees: []
milestone: M5
parent: null
origin: null
dependencies:
  - TKT-01M62BMD18K3SE1WXJ7FDN8EPT
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:49Z
updated_at: 2026-11-02T15:59:16Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
extensions: {}
---

## Description

The perception scan reads colliders into a buffer of 2048. In a big base,
more colliders than that inside the radius means some objects are missed,
and the scan says it was truncated. Accepted in PR 3. Fix if the trace shows
truncation at home: a larger or growing buffer, or a smaller radius for
structures.

### Why it matters to M5

M5's upkeep goals (docs/ROADMAP.md, goals 5.1 to 5.3) depend on perception seeing every fire, cooking station and chest at home. A truncated scan can hide a fire that went out, which the soak report would then count as unobserved rather than failed. The upkeep report (TKT-01M6EFK453 (VM: base upkeep report from a home soak audit window)) lists perception `scan` traces with `truncated: true`; check that first.

### What exists

- `Perceiver.cs` uses a fixed `Collider[2048]` with `Physics.OverlapSphereNonAlloc` for resources, and its 30 s `scan` trace reports `truncated` and `mostColliders` (plugin/Renki.Plugin/Perceiver.cs).
- A growing buffer already exists and is tested: `ScanBuffer<T>` in plugin/Renki.Core/Perception/ScanBuffer.cs (with plugin/Renki.Tests/ScanBufferTests.cs), used by HarvestTask.cs for its safety scan (256 growing to 2048). It treats a full buffer at the maximum as a failed, not complete, query.

### What to build

Use `ScanBuffer<Collider>` in the perception scan, starting at 2048 and growing to 8192, and keep reporting `truncated` when the maximum is reached. Keep the scan within its time budget (the `scan` trace's `avgMs`, 2 ms target).

### Files

- `plugin/Renki.Plugin/Perceiver.cs`
- `plugin/Renki.Core/Perception/ScanBuffer.cs` and `plugin/Renki.Tests/ScanBufferTests.cs`

### Verification

`just check plugin test-core ScanBufferTests` and `just check ci`. Live evidence is no `truncated: true` scan trace in the upkeep soak windows, with `avgMs` at or under 2 ms; read it from the incidents section of `just vm upkeep-report`.

## Acceptance criteria

- [ ] A scan with more colliders than the buffer still reports everything in range, or grows the buffer
