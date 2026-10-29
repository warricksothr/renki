---
schema: 4
id: TKT-01M64ZZYGH64Y9BYGY7BN4E1AQ
title: "Panel: show remembered chest contents on hover"
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
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
created_at: 2026-10-29T23:30:16Z
updated_at: 2026-10-29T23:39:44Z
created_by:
  id: agent:codex/renki-resume
  name: ""
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

Show last-known chest contents on map hover using the same per-world container memory used by the bot. Include quantities, identity and the observation time; distinguish an unopened chest from a chest last observed empty. Keep grouped chests readable and clear remembered contents across world/connection changes. The user explicitly requested implementation while food recovery validation was under way.

## Acceptance criteria

- [x] Chest hover shows quantities and last-observed time from shared memory
- [x] Unknown and observed-empty chests are distinguished
- [x] Grouped markers stay readable and world changes clear old contents

## Implementation plan

Include remembered inventories for mapped chest entities in the existing scoped map snapshot. Reuse the shared store, show bounded per-chest contents and observation times on hover, distinguish unknown and empty, verify memory scope in Go and known/grouped hover behavior in the real-assets browser preview, then land and refresh the controller.

## Notes

**agent:codex/renki-resume** at 2026-10-29T23:33:13Z

MapSnapshot now includes only mapped container inventories from the same per-world store used by bot memory. Hover shows entity/coordinates, quantities aggregated across stacks and the last observation time; it distinguishes unknown from observed empty, limits grouped previews and guides zooming. Existing generation/connection/world reset also clears these inventories. Focused Go checks verify shared-store updates replace contents and other-world/unmapped/menu contents are excluded. Real-assets browser fixture verified known, empty, unknown and four-chest grouped tooltips (within map bounds), reset/reload and no browser errors. just check ci passed; deployment still pending. Added just vm task-check to retain full task evidence, select the correct controller run and report one terminal result without resending. Real plugin success and refusal paths both exercised.

## Summary

Landed in PR #33 (main 3ab1fbe), deployed with verified checksums. Live API snapshot exposes seven remembered inventories from the shared world store. Browser checked the actual captured food chest totals (9 deer meat, 7 neck tails, 2 cooked meat, fish preserved) and an empty chest with matching observation timestamps, plus known/unknown/grouped/reset fixture cases. Full CI and commit review passed. Refresh the panel to load the new hover behavior.
