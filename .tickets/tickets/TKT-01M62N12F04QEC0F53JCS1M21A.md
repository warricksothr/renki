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
updated_at: 2026-10-30T06:34:27Z
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

- [x] harvest fells a tree, splits a log and breaks a stump or rock by id with the best tool it carries
- [ ] It collects the drops within a few metres unless collect is false
- [ ] It fails with no_tool, inventory_full or target_lost as 6.6 names them
- [x] It does not stand under a falling trunk

## Implementation plan

The cb0192a retry positively observed physical strike sight on full Beech1 1:24194, then refused in Protected. The last perception contained only a Deer about37m away and no structures. Protected currently maps both a real collider exclusion and a saturated256-collider all-layer overlap to the same refusal; Settled uses the same fixed buffer and can also wait forever on unrelated dense scenery. Retain exact cause evidence rather than reducing exclusions.

Use a reusable scan buffer that retries full queries at bounded doubling capacities up to2048 (the existing perception cap). Consume only complete results; at the maximum, refuse explicitly as dense scene. Use it for the existing all-layer Protected and Settled scans, preserving their radii/layers and actual collider checks. Keep the item-drop scan separate. Emit bounded phase/capacity/count/completion metadata on expansion/saturation and exact blocking collider/character/piece evidence on a real exclusion. Stop explicitly if settling cannot be fully scanned. Add tests proving a protected object beyond the old truncation remains visible, exact-capacity retry, reuse with smaller result sets and bounded maximum failure. Validate/review/land/deploy and retry autonomous full-tree/log work; the new trace will establish actual density versus actual hazards.

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

**agent:codex/renki-resume** at 2026-10-30T00:08:26Z

Installed 5d51fc8: t-1 again reached the stump from inside home, but final approach pressed against its wider base and recovery jumped onto it. Owner screenshot independently confirmed the behavior. Bounded contact trace measured axe range2.2, attackHeight0.8, selected reach1.0 and physical contact [-30.22,33.01,33.2]; terrain stance [-30.52,32.48,32.22] was just outside that unnecessarily restrictive radius. Once above the stump, horizontal-only look also missed the low target. Task failed unreachable; t-2 returned home successfully and just vm stop completed. No products or success claimed. Evidence .checks/task-didvylz4.json, .checks/audit-window-JMurji.jsonl, .checks/audit-window-Eocs7d.jsonl, .checks/task-0jwlx1qp.json. Implemented range-scaled stance with a 1.4m conservative maximum (still within 2.2m axe reach), downward normal attack aim from tool attack height, and disabled jump/climb recovery within3m of the resource. Added actual-coordinate regression and honest suppressed-jump trace. Retrying live after final gate/review/deploy.

**agent:codex/renki-resume** at 2026-10-30T00:17:50Z

PR #38 installed as 4f50bb9; deployment hashes verified. Final approach stayed on terrain beside stump 1:39764; t-1 swung with axe wear97→96, then stopped when Nimah entered the character exclusion radius. No successful stump destruction/collection is claimed. Evidence .checks/task-hfu96y62.json and .checks/audit-window-kbls0F.jsonl. Necks subsequently appeared; automatic flee superseded the return attempt and reached25m safety. Minimum health93.6 before food regeneration; logged out and stopped, rejoined after owner offered to clear the site, and t-1 return home passed (.checks/task-ub7udn16.json).

Owner requested PvP-aware proximity: verified ordinary attack filtering plus victim damage rejection in installed game source. Implement nullable PvP observations for self/nearby players and relax player exclusion only for stationary stumps/small trees with normal tools, valid known PvP on both players and at least one disabled. Keep unknown/both-enabled, animals, pieces, full trees/logs and mining protected. Truth-table regressions and explicit false/true shared fixtures added. just check ci passed (20261030T001656Z-NmDT0n.log). Live PvP guard and full from-home delivery loop remain pending.

**agent:codex/renki-resume** at 2026-10-30T00:36:46Z

Build a1a1a41 completed the unstaged raised-home stump benchmark on 2026-10-30 UTC. t-6 started from [-39.56,36.09,7.02] inside home, approached Beech_Stub 1:39764 on supported outdoor terrain, landed four attributed hits, broke it and collected 2 new Wood. Final resource stance [-30.55,32.49,32.06] remained beside the stump; the former jump-on-stump failure did not recur. Axe durability96→92. t-7 returned to the saved home storage waypoint; t-8 deposited exactly the 2 new Wood into1:30595 (chest67→69); t-9 returned home. Original Wood2, BeechSeeds15 and AxeFlint1 remain. Final health112/112, sheltered at [-37.56,36.09,7.3], no active task. Health tracked declining food maximum; no damage event in this window. Active meals refreshed automatically earlier, consuming the three previously carried spares normally.

Receipts .checks/task-3kbvjvot.json, .checks/task-68smxtw6.json, .checks/task-xrezsg2b.json and .checks/task-zqclusoo.json. Exact full loop .checks/audit-window-8he1ZH.jsonl. The initial house exit still needed Jump/Strafe/Repath recovery around [-41.3,36.09,12.4] before the successful terrain approach, so this is a positive bounded check rather than a claim of ideal navigation. It used no manual outdoor staging.

Own PvP false is observed; no nearby Nimah was present, so the protected-bystander live check remains open. Full-tree falling escape, descendant log splitting, rock mining, collect:false live behavior and the complete compound criteria are still unverified. Do not reuse destroyed stump1:39764. The other stump1:39765 was still present in the last live map.

**agent:codex/renki-resume** at 2026-10-30T05:43:13Z

Added just vm harvest-trial for one owner-independent full Beech1 and newly observed nearby same-family logs, using normal native harvest goals. It shares startup, saved observations, task receipts, exact audit-window capture and unconditional logout/stop/combat-off cleanup with the combat supervisor. It never issues movement tasks. Before cutting, require active food/80% HP/40 stamina, a usable supplied axe, four spare slots,60 spare carrying weight and all-character/structure clearance. Scope every observation and inventory snapshot to the same live world/connection; abort on visible health loss, takeover, unsafe/unavailable logs and travel/time/task limits.

A positive benchmark requires matching native local-hit/break evidence for each successful task, at least two new log harvests, collected Wood confirmed by exact pack delta, and no observed unfinished new nearby beech logs. Initial log IDs remain excluded. Proximity does not prove spawn lineage. Five tasks/eight minutes/40m bound the attempt. Does not claim mining, crafting, home delivery or indefinite harvesting.

Added authenticated read-only GET /api/inventory with one-lock live world/connection/weight/pack scope; unknown/menu/disconnected inventory is never presented as current. This avoids downloading broad audit history just to verify pack counts. Focused controller session/panel/app checks passed (.checks/20261030T053749Z-moxuXl.log). just check vm test-harvest-trial passed (.checks/20261030T053902Z-75yVzI.log); full just check ci passed including 323 plugin tests, five combat supervisor tests and four harvest supervisor tests with nine integration scenarios (.checks/20261030T054200Z-4eMCez.log). Remaining review/landing/deployment/live trial follow. Harvest acceptance criteria remain open until actual evidence earns them.

**agent:codex/renki-resume** at 2026-10-30T05:54:00Z

PR #58 landed as e36e366, and just check vm install verified package hashes (.checks/20261030T054425Z-VkEzpH.log). First autonomous just check vm harvest-trial --wait-s60 failed timeout after native180s without a hit (.checks/20261030T054510Z-L0OtQs.log). Tree Beech1 1:24194 at [175.55,84.17,-250.77]; initial pack AxeFlint1/BeechSeeds15/Wood2. Retained observations/approach trace put feet at [175.83,84.45,-249.43], measured contact [176.21,84.97,-250.22], attackHeight0.8 and reach1.4: physical strike range already passed, but generic UseReach.Near could not see the tall nonconvex tree through its overhead bounds center. Walker kept trying a stance only0.14m away. No damaged event or cleanup error occurred. Summary .checks/harvest-trial-20261030T054510700197Z/summary.json; audit .checks/audit-window-fk9shF.jsonl.

Harvesting now tests sight from the axe-height origin to the measured physical contact using normal terrain-aware attack layers and first-solid-target policy. Self/triggers are ignored, saturated queries refuse, and actual occlusion at an in-range stance fails explicitly rather than initiating another endless approach. Generic chest/door interaction sight is unchanged. A first-visible-contact trace retains exact origin/contact/range for the next attempt. Physical reach/aim/threat guards and falling escape stay in place.

Focused just check plugin test-core HarvestTests passed18 cases (.checks/20261030T055132Z-qRSq8W.log). Initial CI caught only the later trace initializer's whitespace; corrected before final just check ci passed with325 plugin tests, five combat and four harvest supervisor tests (.checks/20261030T055319Z-tCKLIH.log). Review/landing/deployment and the same autonomous benchmark follow. No acceptance criterion was checked from approach-only evidence.

**agent:codex/renki-resume** at 2026-10-30T05:56:22Z

Before retrying, aligned the benchmark's candidate exclusion with the native full-tree fall guard's maximum32m radius. Settled logs use8m rather than inheriting a standing tree's fall radius;15m around the bot remains clear. This reduces predictable refused candidates while preserving native guards. Added a regression distinguishing a creature25m away from full-tree versus log work. Final just check ci passed325 plugin tests, five combat and four harvest supervisor tests (.checks/20261030T055533Z-AyIAtv.log). The already committed strike-sight fix reviewed without findings; review the combined commits again before pushing this follow-up.

**agent:codex/renki-resume** at 2026-10-30T06:08:48Z

The cb0192a retry passed physical strike sight but refused before a swing. Evidence: .checks/harvest-trial-20261030T055804421383Z/summary.json and .checks/audit-window-fsiaZG.jsonl. That build conflates a full 256-collider buffer with an actual character/structure exclusion; saturation is a hypothesis until new trace evidence establishes the cause.

Implemented complete-result safety scans with reusable 256-to-2048 bounded growth, explicit dense-scene refusal and exact blocker traces, preserving protection/settling radii, layers and PvP rules. Focused ScanBufferTests passed (.checks/20261030T060427Z-Hdd7nT.log). Full just check ci passed (.checks/20261030T060547Z-Xhdxv1.log). Review, landing, deployment and autonomous retry follow; no additional acceptance criterion is verified yet.

**agent:codex/renki-resume** at 2026-10-30T06:13:41Z

PR #60 installed as 53a27dc; just check vm harvest-trial --wait-s 60 positively felled full Beech1 1:24194 after seven local hits. Safety trace establishes old-buffer saturation: protection radius32m, 281 complete colliders after256-to512 growth, no blocker. Tas moved from the trunk to about[177.25,84.14,-244.98] before returning for collection; seeds15->17 and Feathers0->2. All observed HP equalled decaying maximum80.9->80.4; no damaged event or cleanup error. Subsequent native harvest of newly observed beech_log1462070434:49 at[175.46,86.65,-265.53] failed unreachable/no physical contact before approach. No log split or Wood gain is claimed. Evidence .checks/harvest-trial-20261030T060943568520Z/summary.json and .checks/audit-window-LxVrpf.jsonl; installation .checks/20261030T060931Z-TXC7sB.log.

Next preserve the successful standing-tree contact probe and add a full3D collider-centre ray fallback for tilted logs whose horizontal pivot-height slice is empty. Actual Collider.Raycast hit, fresh strike visibility, ground support and all safety guards remain mandatory. Add a log-only autonomous retry mode to use the fallen trunk without felling another tree, explicitly reporting partial log coverage rather than full-tree benchmark success.

**agent:codex/renki-resume** at 2026-10-30T06:19:47Z

PR #61 installed b0d4538 and log-only trial passed physical contact acquisition/approach on the tilted beech_log1462070434:49. Contact[175.61,86.71,-264.56], strike origin[175.83,86.8,-263.24], reach1.4. It then refused its one outward escape before swinging. No damage/cleanup errors; minimumHP79.8 equalled current food maximum. Evidence .checks/harvest-trial-20261030T061605556575Z/summary.json and .checks/audit-window-VVI6C2.jsonl. Log splitting remains unverified.

Add up to seven outward six-metre escape candidates, alternating30-degree offsets through90 degrees, one candidate per physics tick. Each corridor moves monotonically away from the resource, validates support/water/wall/body clearance every half metre and retains physical refusal diagnostics; no candidate means refusal. Tests exercise the actual failed log coordinates and prove distance monotonicity, bounded travel and invalid input refusal. Also automate bounded harvest evidence projection in trial summaries and just vm harvest-evidence, so future iterations read exact matching contact/scan/escape/break details without broad logs. Raw audits remain retained.

**agent:codex/renki-resume** at 2026-10-30T06:34:15Z

39b0b2c log-only trial PASSED: beech_log1462070434:49 split after5hits, new beech_log_half1761239627:42 broke after6hits and1761239627:43 after4hits. Original escape candidates1/2 hit Bush01/model; candidate3 passed and was used. Wood2->22, original seeds17/feathers2 preserved, no unfinished observed nearby logs. No damaged events or cleanup errors; minimumHP76.7 tracked food maximum. Summary .checks/harvest-trial-20261030T062121409419Z/summary.json; audit .checks/audit-window-4KNHu1.jsonl; wrapper .checks/20261030T062121Z-zj7uf5.log. Together with prior supplied-axe stump and full-tree trials this verifies tree/log/stump operation by ID and falling escape on the exercised cases. Compound collection:false and inventory-full/target-loss branches remain open.

23248bb fresh full-tree trial felled Beech1 1:24184 after6hits and escaped via the first validated corridor, without damaged events or cleanup errors. It then failed the new log's single ground stance: no supported, water-safe position beside resource. This is NOT a complete continuous tree-to-Wood pass. Summary .checks/harvest-trial-20261030T062610751657Z/summary.json. Next geometry improvement should try bounded alternative supported stances around a tilted log rather than its single observer-facing contact, retaining actual strike reach/sight and obstacle checks. The compact automatic harvest evidence identified this without broad history dumps.
