---
schema: 4
id: TKT-01M62N25826TRP761PYVVC7RKY
title: "Plugin: attack task with blocking and dodging"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - safety
assignees: []
milestone: M3
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T01:40:45Z
updated_at: 2026-10-30T00:39:51Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

`attack` (7.2): melee or ranged against an entity, with blocking and dodging, behind Safety.AllowCombat. Fails with target_lost when the target is gone, and stops on the dead-man rule like every body task.

## Acceptance criteria

- [ ] attack fights an entity until it dies or the task is stopped
- [ ] It blocks and dodges, and is refused unless AllowCombat is on

## Implementation plan

First combat foundation after the successful unstaged stump benchmark: add explicit attack protocol arguments/fixtures and a bounded plugin melee task behind AllowCombat plus allowedTasks. Start verification with one unstarred Greyling. Reuse normal aiming/controls and supported approach, but select weapons from character combat damage rather than harvest damage. Recheck target identity, hostility/taming, bystanders/structures, weapon durability, stamina reserve and attack reach. Add held block and guarded single-step dodge controls with complete cleanup on stop, replacement, death, dead-man and disconnect. Verify how remote creature attack timing is actually exposed; unknown animation state must not become a fabricated parry cue. Use supported dodge movement and a bounded encounter leash/progress timeout. Confirm death from live evidence; disappearance is target_lost. Return a clear retreat-needed outcome when safe combat conditions fail. Ranged/auto-style completeness follows the melee slice; broad existing criteria stay open until met. See docs/SELF_DEFENSE.md for arbitration and verification. Keep this ticket draft during design; no combat is enabled by this plan.
