---
schema: 4
id: TKT-01M6EWAJ0N0FW74EH1W7QPB34Y
title: "Perception: report the scan origin and a truncated scan"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - protocol
  - controller
assignees: []
milestone: M2
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T19:38:34Z
updated_at: 2026-11-02T19:38:34Z
created_by:
  id: agent:claude-code/renki-m2-drops
  name: ""
updated_by:
  id: agent:claude-code/renki-m2-drops
  name: ""
extensions: {}
---

## Description

The drop-pile tracker (TKT-01M6EFHE67, Controller: sum drops per cell and prefab with their trend, `internal/piles`) decides what ground perception has seen, so the deliver-when-gone timer (TKT-01M6EFHE4E) never starts on ground the bot could not see. Two things the perception message does not say force it to guess:

- **Scan origin.** `Perceiver.Scan` scans from `me.transform.position`, but `protocol.Perception` carries no position. The controller pairs a frame with a state at most 1 s newer and shrinks the reach by 15 m/s × lag (twice that for a full list), which costs reach and is still an assumption about speed.
- **Truncation.** `Physics.OverlapSphereNonAlloc` fills a fixed 2048-collider buffer across every resource layer. When it fills, drops can be missed with only an audit trace ("truncated"); the message looks complete, and the tracker would read the missing drops as cleared ground.

### What to build

- Add the scan origin (`pos`) and a `truncated` flag (or the hit count against the buffer size) to the perception message, behind a feature so older plugins still work.
- In `internal/piles`, use the origin instead of the paired state and drift, and give a truncated frame no reach.

## Acceptance criteria

- [ ] Perception carries its scan origin and whether the collider buffer filled, in a plugin test
- [ ] The pile tracker uses the origin and treats a truncated frame as seeing nothing, in a test
