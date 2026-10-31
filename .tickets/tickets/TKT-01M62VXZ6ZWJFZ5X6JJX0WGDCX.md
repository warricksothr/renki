---
schema: 4
id: TKT-01M62VXZ6ZWJFZ5X6JJX0WGDCX
title: "Controller: tend template: cooking, fermenters, beehives, ore"
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: null
parent: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
origin: null
dependencies:
  - TKT-01M62VXYSN4RQV75HTA3BSD162
  - TKT-01M62VXZ4D32W7WBX7WV84DAVY
  - TKT-01M62VXYWJHB5BC9VGCBM0PVXD
  - TKT-01M62VXYY6F094WXQS3FBQ57RF
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-milestones
  branch: t3/cooking-food
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: e9bfe1ea794eb8b8fe80c7ecc99a00235cc6b051
  session: null
  claimed_at: 2026-10-31T16:33:34Z
  expires_at: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-10-31T17:05:06Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

A template on the behaviour engine for stations that hold their output or
need feeding:

- Take food off cooking stations before it burns. This is time-critical, so
  the instance gets a high priority.
- Collect from fermenters and beehives when they are ready.
- Bring ore or wood from the chests to the smelters and kilns that take it.

What it collects goes to `stow`.

## Acceptance criteria

- [ ] Cooked food comes off before it burns while the bot is in the base
- [ ] Ready fermenters and beehives are emptied and the goods stowed
- [ ] Ore in the chests is carried to a smelter that takes it

## Implementation plan

Continue the user-authorized food-resupply/escort work with the first cooking slice. Add optional scoped cooking-slot and fire fuel/lit telemetry, sourced from current station/ZDO state rather than cached geometry. Extend interact with an exact carried raw-food prefab and implement one native cancellable cook/wait/collect body: basic fire-backed stations only, empty on admission, live recipe/heat/access/reach/capacity checks, one verified input insertion, no repeated consumption on uncertain RPC outcome, time-critical collection, and measured inventory receipt. Reject foreign occupied slots rather than collecting someone else's batch. Use normal game interactions and existing pickup machinery with both native/session child permissions. Refuse missing ingredients or extinguished fires with actionable diagnostics; automatic fueling and broad tend/storage composition remain follow-ups. Keep wider fermenter/beehive/ore AC open.

Add policy, protocol, permissions, cancellation and receipt regressions, document the task and partial/cancel semantics, run focused recipes and just check ci, commit/review and open a stacked PR above130. Deploy while Tas is offline, then use continuous guard/supervision to inspect nearby safe supplies/stations. Only attempt an actual batch if real carried or safely retrievable ingredients and a working station are observed. Log out/stop on unsafe health, uncertainty or missing prerequisites; record honest live outcomes without claiming full M2/M3 completion.

## Notes

**agent:codex/renki-milestones** at 2026-10-31T16:51:23Z

Implemented the first native cooking slice on t3/cooking-food, stacked above PR130. Owners/operators can request cook food; fresh carried raw boar/deer meat or neck tail and an empty heated station within12m are required. Native interact cook uses normal item/station use and pickup, current recipe/access/safety/capacity gates, one owned input-slot receipt and one removal attempt. Scoped normal SpawnItem and Humanoid.Pickup hooks tie spawned instances and actual pickup inventory gains to this batch. Uncertain insertion/removal is never repeated; loss, foreign slots, burning or interruption fail without fabricated success. Generic pickup merge substitution is disabled for this provenance-sensitive child. Native food consumption of the output waits until its receipt completes.

Added current fire fuel/lit and cooking slot/heat telemetry, shared fixtures, protocol/policy/permission/cancellation/command tests, docs/COOKING.md, and just vm cooking-check. The live-check recipe can join under the enabled native startup guard, inspect unique current home chests for one raw ingredient using verified receipts, and always logs out/stops after admission while preserving panel/video. It prints compact results and retains full evidence. Ordinary food resupply candidate defaults are unchanged; the helper now optionally supports an explicit ingredient allowlist. Ghost-only memories cannot inflate the cooking search set.

Focused cooking core checks, controller protocol/chat checks, native build, cooking supervisor and existing resupply supervisor checks passed. Full just check ci is running. No live cooking success yet. This is a single explicit batch primitive and proof tool, not the complete automatic tending/resupply/escort-maintenance composition. Fermenter/beehive/ore AC stay open, as does live proof of avoiding burnt food.

**agent:codex/renki-milestones** at 2026-10-31T17:05:06Z

The first guarded live proof on fd2ac62 safely withdrew one RawMeat from chest 1:70031 with matched native container/inventory receipts. The supervisor then falsely timed out because it matched task response id rather than the protocol's re field. Raw failed evidence is retained in .checks/cooking-20261031T165430Z; no cooking was attempted. Corrected the parser and added a regression against the shared native task_failed fixture.

Both current nearby cooking stations (1:70085 and 1:70084) were empty but cold; their supporting campfire 1:70049 had zero fuel. Added native preparation using one carried Wood through normal Fireplace.UseItem. It matches actual station fire-check points against inactive burning-area geometry, requires a unique empty wood fire, ward access and sight/reach, and confirms wood consumption, synchronized fuel increase and actual station heat. Unconfirmed or blocked fires never trigger another fuel attempt. Chat and proof-tool selection admit cold empty stations only with carried wood. Broader automatic resupply, repeated tending and fermenter/beehive/ore AC remain open.

Focused cooking core, supervisor, chat and protocol checks passed. Full just check ci is in progress. Tas remains logged out inside home, about30.9/30.9 HP, with the retrieved raw meat and carried wood. Native startup recovery guard remains enabled; home.afterOutage remains temporarily false. Next: commit/review/deploy and repeat one continuously supervised batch. Raised-floor reach remains a possible live blocker; sight/access checks will remain enforced.
