---
schema: 4
id: TKT-01M66SXGAMHZYATMRAP7B5ADSA
title: Report menu after a requested logout finishes
type: bug
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - vm
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-milestones
  branch: t3/m2-m3-completion
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 9117f039f899b68984511cd89fa623dc59eccee0
  session: null
  claimed_at: 2026-10-30T16:47:16Z
  expires_at: null
archive: null
created_at: 2026-10-30T16:22:34Z
updated_at: 2026-10-30T16:48:17Z
created_by:
  id: agent:codex/renki-atlas
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

During atlas restart checks just vm stop took about 68 seconds despite eventual successful game/controller shutdown. Code inspection found RenkiPlugin.Phase returns logging_out before checking whether Player.m_localPlayer and ZNet.instance have disappeared; loggingOut stays true after a requested logout. The VM stop script waits for menu, so an already completed native logout may exhaust its 60-second wait. Confirm this with a focused live trace, then make menu observable after logout without re-enabling automatic rejoin. This issue did not prevent atlas retention/offline checks.

## Acceptance criteria

- [ ] Phase reports logging_out only during shutdown and menu once the game has returned there
- [ ] AutoJoin suppression remains active after an explicit logout
- [ ] A live logout/stop check recognizes menu without exhausting the 60-second wait

## Implementation plan

Publish phase transitions after explicit logout even though body control has ended. Resolve menu once Player and ZNet are gone before returning logging_out; keep the logout latch and AutoJoin suppression, including across a new welcome, so final reporting cannot restore automation. Verify the transition policy and controller handling, then confirm native logout reaches menu without the 60-second wait. This addresses the final-state issue also described by TKT-01M64Z4P80P9W2QSGQ22QAE56W — Logout: publish an accurate final connection and phase state; close that duplicate only after the shared behavior is verified.
