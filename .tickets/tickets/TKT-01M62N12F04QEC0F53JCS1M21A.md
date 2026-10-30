---
schema: 4
id: TKT-01M62N12F04QEC0F53JCS1M21A
title: "Plugin: harvest task: chop, split and break with the best tool"
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - movement
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62N129K1RGB8VSJZ3SDAC7N
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-resume
  branch: t3/harvesting
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 9f47b78ff635a07b66753ed259f564dc8b97b080
  session: null
  claimed_at: 2026-10-29T22:56:16Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-30T00:03:46Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

`harvest` (7.2): approach a tree, log, stump or rock by entity id, equip the best tool that meets its `minTool`, face it, and swing until it breaks; with `collect` (the default), sweep up the drops within a few metres afterwards.

A felled tree becomes a new `log` object and a `stump`; a log splits into halves, each a new object, and those break into wood (6.2). One harvest is one object; the controller chains them by watching perception for what appeared where the old one was. Swinging needs care with stamina, and with the falling trunk, which hurts.

Fails with `no_tool` when nothing in the inventory meets minTool, `inventory_full` when collecting cannot fit more, and `target_lost` when someone else breaks it first.

## Acceptance criteria

- [ ] harvest fells a tree, splits a log and breaks a stump or rock by id with the best tool it carries
- [ ] It collects the drops within a few metres unless collect is false
- [ ] It fails with no_tool, inventory_full or target_lost as 6.6 names them
- [ ] It does not stand under a falling trunk

## Implementation plan

Replace collider-bounds proximity with real collider contact at tool attack height. Pick a stand-off beside the actual contact, project onto measured support near the resource's own level, and pass a tight grounded goal to the existing wall/water/clearance-validated walker. Never inherit the raised starting floor as the outdoor resource's destination height. Recompute bounded approaches after missed swings, using observed hit evidence for progress and retaining injury/threat/protection/settling/escape checks. Cover the recorded raised-house and too-far small-tree cases offline, then verify a from-home harvest/collect/deliver/return with an unmodified physical target.

## Notes

**agent:codex/renki-resume** at 2026-10-29T23:18:25Z

Live supplied-tool trial on 4831c63, 2026-10-29 23:07–23:10Z: mineable rock 1:31313 correctly rejected no_tool (no carried pickaxe). Reached Beech_small1 1:39689; normal axe input physically broke it, durability 100→99, actual pack gain Wood 2 with BeechSeeds 15 and AxeFlint 1 retained. A Greyling approached during the trial; harvest t-4 refused when its near-character check fired before normal attack animation completed. Therefore this is physical swing/drop evidence, NOT a successful harvest task or completed delivery benchmark. Do not check acceptance boxes from it. Evidence .checks/audit-window-DYfedw.jsonl and .checks/audit-window-a69Odi.jsonl.

Tas lost health during this encounter and another Greyling on return. Sent escapes and return routing; last observed 2.7/25 HP at (-32.46,33.58,-5.33). Requested logout, then just vm stop completed and just vm status confirmed game/controller stopped. Axe 99/100, seeds 15 and wood 2 saved; no death observed. Owner has since confirmed the route/site are safe and food is available in chests. Recovery/eating is needed before another cut.

Source hardening: refuse harvest below 80% HP or with an enemy within 15 m, recheck enemies every .25 s before approach/collection, cancel held attack/pickup immediately on health loss. Moving-log checks also require two seconds of quiet observed pose, including remotely simulated motion and accumulated slow drift; do not trust zero local rigidbody velocity alone. Escape is a one-time phase so later collection can approach settled drops. These fixes still require live validation. No full tree/log/stump/mining success, inventory-full branch, controller descendant chain or completed chop/deliver/return benchmark is claimed.

**agent:codex/renki-resume** at 2026-10-29T23:34:20Z

On installed PR #32, Tas recovered naturally to full health from the shared food chest. Harvest t-14 inside and t-17 on the west terrace both refused the 15 m threat guard while current character perception showed no hostiles. Code inspection found that the all-layer overlap returns the same enemy result when its 256-collider buffer saturates; dense house geometry can therefore masquerade as a threat. Replace that scan with the same Character.GetCharactersInRange registry already used by perception, retaining all enemy/distance checks and 0.25 s cadence. No threat safety is bypassed. Full supplied-tool task/collection/delivery, log/falling-tree safety and post-fix live retreat remain unverified. Current pack still contains the original axe, 15 seeds, two wood and one spare of each of three meals.

**agent:codex/renki-resume** at 2026-10-29T23:39:35Z

PR #33 landed as main 3ab1fbe and just check vm deploy passed with checksums. New run harvest t-1 accepted (character-registry threat scan passed) but cancelled 0.1 s later as its health baseline fell with food maximum decay. Audit .checks/audit-window-KfUlis.jsonl contains no damage events; Tas stayed at home at full current HP. Clamp the fallback health baseline to current maximum so ordinary food decay is not damage; actual positive damage still latches cancellation immediately even when healing masks a loss. Add a numerical regression for food-cap decay versus real injury. Supplied-tool benchmark remains open pending this correction.

**agent:codex/renki-resume** at 2026-10-29T23:48:10Z

Final installed build main 0eea488 (PR #34): just check vm deploy passed with package/plugin hashes. Food-cap decay no longer cancels harvesting, and the character-registry guard no longer refuses dense scenery.

Live results in the controller run starting around 23:42Z: t-1 from the raised house failed its supported-route search; its target surface inherited the house-level approach height. Staging west outside via t-2 passed, but t-3 stopped after eight seconds without a damaging swing from its chosen reach. Do not call automatic approach complete. Explicit ground approach t-5 to [-26.5,32.57,31.7] passed. From that staged position, t-6 harvest on Beech_small1 1:39689 succeeded in 7.35 s: two observed local hits, axe wear 99→97, pack gain Wood2 and Resin2. No competing hit or hostile damage was observed. Pack changed Wood2→4, Resin0→2; all 15 seeds and spare foods were preserved.

T-7 returned to the accessible inside-floor waypoint using the learned route (161 validation samples, 5.51 ms total, .33 ms worst slice). T-8 deposited exactly the two new wood and two resin into chest 1:30595. The fresh before-transfer observation held Wood65 (newer than the old Wood32 memory); after it held Wood67 and Resin2. No existing items were removed. T-9 returned home at [-37.23,36.09,7.85], with current HP equal to its normally decaying food maximum and no task active. Original Wood2, axe, seeds15 and one spare of each meal remain carried.

Evidence: .checks/task-b7osuq4j.json (raised approach), .checks/task-j6x38emt.json (reach failure), .checks/task-cwikgxlr.json (positive harvest), .checks/audit-window-vnMcd3.jsonl (hits/products/wear), .checks/task-k0drjzfs.json (deposit), .checks/audit-window-ztwBYr.jsonl (conservation/return).

This is a successful staged small-tree chop/collect/deliver/return slice, not closure of the compound harvest criteria. Next: ground-supported resource approach from raised floors, reliable attack contact rather than broad collider bounds, full tree/log/stump/rock cases, collect:false, inventory-full/target-loss and live post-fix retreat. All harvest criteria remain unticked until their whole scope is verified. Automatic food upkeep and remaining storage capacity/ward/overflow cases remain on their own tickets.

**agent:codex/renki-resume** at 2026-10-29T23:58:14Z

Implemented physical Collider.Raycast contact at attack height, support projection anchored at resource elevation, tight stand-off approach and one bounded closer retry after missed swings. Added captured-coordinate regressions for raised home versus outdoor support and actual strike reach. just check ci passed (20261029T235737Z-Wj8non.log), including both protocol fixture suites and plugin build/tests. Not yet deployed or live verified; all compound harvest criteria remain open. Next live check must begin inside raised home without manual staging, then collect/deposit only measured gains and return.

**agent:codex/renki-resume** at 2026-10-30T00:03:46Z

Installed dc981d1: t-5 harvest from raised home reached outdoor stump 1:39764 directly without manual staging and without health loss, but never swung. Evidence .checks/task-9fx2jr97.json and .checks/audit-window-gZ6auq.jsonl: navmesh follower stopped within its default final half-metre tolerance while harvesting still needed a tighter physical-contact stance. Cancelled with t-6 return-home, which passed, then just vm stop succeeded. Corrected PathFollower with an optional tighter final tolerance, wired from Walker arrivalWithin; intermediate waypoints and partial-path ends retain existing semantics. Added a regression using the actual stopped position plus bounded contact/goal/tool-range traces. just check ci passed (20261030T000304Z-mQg7u4.log, 198 plugin tests). Retrying live after review/deploy; no stump success is claimed yet.
