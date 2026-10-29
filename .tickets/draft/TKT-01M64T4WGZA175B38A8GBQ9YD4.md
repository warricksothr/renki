---
schema: 4
id: TKT-01M64T4WGZA175B38A8GBQ9YD4
title: "Tools: acquire, craft, repair and upgrade harvesting equipment"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - plugin
assignees: []
milestone: null
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T21:48:07Z
updated_at: 2026-10-29T23:06:49Z
created_by:
  id: agent:codex/renki-resume
  name: ""
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

After the collection-and-delivery loop and single-target harvesting, give Tas a tool-readiness plan for gathering jobs. The owner requested crafting when no suitable axe exists, reporting missing supplies, and upgrading when resources become available.

Prefer a suitable unbroken carried tool, then an owner-designated stored tool. Repair at an accessible appropriate station when needed. If none exists, plan a known recipe against live/remembered supplies, station type/level and game progression. Gather available prerequisites with supported jobs, fetch permitted materials, and report exact shortages or an unavailable station/unlock instead of promising an action the body cannot perform. Do not loop forever on unobtainable inputs or bypass the game's recipe/unlock requirements.

Craft and upgrade through game APIs with exact before/after item counts. Upgrades should use an explicit owner policy (permitted tool tier, materials, reserves and spending limits) rather than consume arbitrary shared stock because it is present. Resume the original gathering job only when the tool is ready. Owner stop, retreat, death and reconnect invalidate all remaining steps. Crafting is a follow-up to M2's supplied-tool benchmark, not a prerequisite for loose flint/stone collection.

## Acceptance criteria

- [ ] Missing or broken tools produce an actionable recipe, station and material shortfall
- [ ] Suitable carried or designated stored tools and repairs are preferred over crafting replacements
- [ ] Crafting and upgrades obey game progression and an explicit resource policy
- [ ] The gathering job resumes with a usable tool; stop and survival interruptions cancel the plan

## Notes

**agent:codex/renki-resume** at 2026-10-29T23:06:49Z

Supplied-tool baseline established during the approved harvesting slice: owner placed a new flint axe in rag chest 1:30592. Withdrew exactly AxeFlint 1 and retained its other stock (Stone 1, Flint 3, ArmorRagsChest 1). Pack before work: quality-1 AxeFlint at durability 100/100 and BeechSeeds 15. Evidence .checks/audit-window-Nw78Tk.jsonl. New HarvestTool policy selects eligible carried resource damage, tier and positive durability rather than guessing from prefab name; broken/inappropriate tools report no_tool with required type/tier. This does not yet fetch stored tools automatically, repair, describe material shortages, craft or upgrade, so all tool-readiness acceptance boxes remain open. The supplied axe is benchmark equipment and is not authorization to spend unrelated shared stock on upgrades. Derive recipes/station requirements from live game data in the next slice; keep an explicit upgrade/reserve policy before spending.
