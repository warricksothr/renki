---
schema: 4
id: TKT-01M62N12JW8TFN12MAT8M1ADZ8
title: "Controller: go chop wood: pick a site and gather on its own"
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62N12F04QEC0F53JCS1M21A
  - TKT-01M62N12DNF3RFVWAQMK1R9X4J
  - TKT-01M62N12HJW32JJTD2M3PEAEA4
  - TKT-01M62N12G98VW2CYW94FG5RCTB
  - TKT-01M62VXYSN4RQV75HTA3BSD162
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-milestones
  branch: t3/multi-tree-home-storage
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: b704d7318813c104f6e7bbaf1516192079eae5e6
  session: null
  claimed_at: 2026-10-31T05:33:39Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-31T05:50:19Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Scene 2 of the M2 benchmark. On "go chop wood", the bot chooses where: the most recent work site it remembers for this world, or else the nearest trees in memory and perception within a sensible distance of home. It carries out harvest on tree, then the log halves, then the stumps if worth it, collects the wood, and hauls it home as the hauling routine does when full. It stops when told, when it has no axe, or when it has hauled a set amount.

No-chop zones (places to leave alone) come later and are filed separately; the choice of site should leave room for them.

## Acceptance criteria

- [ ] Told to chop wood, the bot goes to the last work site, or else the nearest trees it knows
- [ ] It fells, splits and collects, and hauls home when full
- [ ] It says why it cannot (no axe, no trees known) and stops when told

## Implementation plan

Complete the owner's explicit autonomous multi-harvest -> home -> verified storage objective. The gate/review/checksum-deployed door correction in merged PR128 passed an isolated staircase-to-home return in 5.9 seconds through the two current gates; exact food resupply and indoor floor were verified with zero damage (.checks/resupply-trial-20261031T052940675235Z/summary.json). Directly-under-floor routing is still unproven.

Extend the existing gather runner with an explicit bounded tree count (legacy definitions remain one tree), finish each tree's observed new-log chain before selecting another, and preserve one initial product baseline for gains-only delivery. Check capacity and axe durability between complete trees; these pressures choose storage rather than abandoning completed cargo. Pin known home for an opt-in return-home phase before storage, use native dry movement for work/home approaches, and verify arrival from a subsequent state before depositing. Keep health, bystander, structures/falling-tree, freshness, ownership, failure and uncertainty guards. Bound per-tree work, total native cuts, and reserve logistics time.

Update the owner command and repeatable VM benchmark for two distinct full trees and their new logs, with exact task-matched break/collection/storage proof and final home pose. Select/approach safe sites autonomously; diagnose changed-layout route failures from recorded next-waypoint/physics evidence and fix the underlying behavior without extending return timeouts or asking for manual staging. Focused checks, full CI, committed review, deployment and iterative changed-build live tests precede landing and acceptance updates. Repair/resupply only as actual prerequisites require; leave broader production-pressure/value/armour policy draft.

## Notes

**agent:claude-code/renki-groom** at 2026-10-29T03:40:56Z

2026-10-28: build this as a gather template on the behaviour engine (TKT-01M62VXYSN4, 'Controller: behaviour engine: templates, instances, triggers, repeat'), so 'go chop wood' starts an instance with an until condition, and stock targets (TKT-01M62VXZF8N7) or the agent can start it too.

**agent:codex/renki-resume** at 2026-10-30T06:37:20Z

Implemented bounded gather template: explicit world/site and ordered destination chests; one full tier-zero beech, then only new nearby beech logs; five harvest targets/eight-minute bound; native harvest owns movement/aim/swing/escape/collection. Requires fresh target removal and inventory after native success, at least two log harvests and measured Wood gain before stow. Existing product counts are preserved and only gained Wood/BeechSeeds/Resin/Feathers enter the task-receipt-based stow runner, optionally returning to the site. No default instance is enabled. Unavailable descendants stop the chain rather than selecting another tree.

Focused just check controller test-packages ./internal/behaviour ./internal/app ./internal/session passed (.checks/20261030T063331Z-UJ1mbC.log); subsequent health-loss/food-decay and pre-travel-character guards passed focused behavior checks (.checks/20261030T063553Z-VCsbxg.log). Tests cover complete tree/log/halves/deposit/return, old-log exclusion, preservation of original inventory, unsafe/unavailable descendants, stale success observations, weight measured from current inventory, tools/food/health/stamina/scope/time/target bounds and owner takeover preventing later steps. Final CI/review/landing/deployment follow. Controller-level gathering/delivery, site inference, remembered owner work sites, continuous multi-tree hauling and chat start/stop remain unverified or unimplemented; all broad AC stay open.

**agent:codex/renki-resume** at 2026-10-30T06:45:16Z

PR #64 merged as67e2e1e; bounded gather is installed in cd08433. Final just check ci passed (.checks/20261030T063721Z-QwtwVM.log), committed just check review passed (.checks/20261030T063806Z-tbpNBn.log), and package checksums verified (.checks/20261030T064329Z-WRdP6W.log). No gather instance was installed or enabled. Deterministic tree/log/halves/gain-only-stow/return and owner-takeover checks pass. Controller-level live delivery, site inference, chat controls and repeated multi-tree hauling remain open; component harvesting success does not close these AC. Use docs/BEHAVIOURS.md for current parameters and bounds rather than rereading implementation history.

**agent:codex/renki-milestones** at 2026-10-30T17:46:20Z

claim taken from agent:codex/renki-resume by agent:codex/renki-milestones

**agent:codex/renki-milestones** at 2026-10-30T19:06:21Z

PR #96 merged the broad-site approach regression/fix. Focused behavior checks, just check ci (.checks/20261030T185100Z-mTlVk9.log) and committed just check review (.checks/20261030T185134Z-uPodSM.log) passed. Deployed in 635f4e9 and retained in installed 5a5f5c9; checksum-verified installation .checks/20261030T190549Z-IPaAzz.log. Full controller tree -> logs -> gain-only chest receipt -> return proof remains open, along with site inference/chat and repeated hauling. The previous pending approach stash was removed only after its exact implementation landed.

**agent:codex/renki-milestones** at 2026-10-30T20:22:23Z

Added a controller-owned gather benchmark with exact instance/native step attribution, distinct tree/new-log break proof, measured gain-only chest receipts, preserved starting products and return-site pose/task verification. Supervisor checks cover lost behavior-save responses, bad/missing audit proof, damage between polls and cleanup failure; temporary definitions are stopped/deleted and the game is stopped with combat off and offline panel online. Compact trial-status now finds gather runs and tolerates missing terminal results.

First live attempt against installed a525b18 aborted before any task: no isolated full beech in the native selector's 18 m radius. .checks/gather-trial-20261030T201912076800Z/summary.json and .checks/20261030T201912Z-pBuX0o.log retain evidence. No damage events, no travel/work, final66.2/66.2 HP and successful cleanup. Detailed observations show nearest three trees11.9–13.7 m away were close to a Boar/structures; tree1:61548 was18.48 m away but only30.5 m from the Boar, still inside32 m falling protection. No currently observed full beech passed protection even when considering farther trees. This is a correct refusal, not gathering proof or an owner-clearance request.

The controller benchmark now considers observed sites up to60 m away because gather can approach before its18 m native acquisition; unchanged falling/character/structure guards remain authoritative. Bounded nearest-candidate diagnostics explain subsequent refusals without broad perception dumps. Focused benchmark/native regression checks passed; final CI/review follow. Full delivery, chat/site selection and repeated hauling remain open. Do not rerun the same protected site until a new safe-site acquisition strategy or changed observations justify it.

**agent:codex/renki-milestones** at 2026-10-30T20:31:03Z

Implemented owner/command-profile grammar go chop wood, chop wood and gather wood, routed to the existing controller gather template without model interpretation. The bounded command selects a safe observed tier-zero beech within60 m using remembered danger preferences, or reuses its persisted same-world site when freshly confirmed/distant enough to need travel and revalidation. Home/site remain within150 m of home; ordered storage hints are within10 m of home. Existing native safety, gain-only receipt-based stow and return remain authoritative. Missing prerequisites and active/queued work are explained before starting. One manual once-only Owner wood gathering instance stores the requested site and never resumes automatically after restart; a user-authored name collision is refused.

Focused behavior/chat/app checks passed: .checks/20261030T202955Z-Xx77CT.log. Coverage includes distant approach without relaxed protection, foreign/unsafe-local site exclusion, confirmed remembered-site preference, missing axe/food/capacity/stamina/home/storage/fresh perception, unauthorized chat, permission refusal, user-definition preservation, persisted non-resumption, queued owner stop and a newer stop arriving during planning. The latter binds the queued work to the original control revision so a settings save cannot resurrect an older order. A separate existing player-presence test timed out once in an earlier focused run; its targeted rerun and the subsequent full focused suites passed (.checks/20261030T202804Z-sTD8Ls.log and .checks/20261030T202909Z-tfOVFK.log). No presence change was made merely to hide that timeout.

Final CI/review/landing/deployment follow. All broad M2 AC remain unticked until required live proof is earned. This command is one bounded tree cycle, not repeated multi-tree hauling, mining or crafting. M3 positive kill/block/dodge proof also remains open.

**agent:codex/renki-milestones** at 2026-10-30T21:05:17Z

The live gather trial on28a488f selected full beech1:61548 at[23.62,31.37,33.53],18.24m away, then stopped before harvesting on a false health-loss guard. Summary .checks/gather-trial-20261030T205338134340Z/summary.json and exact audit .checks/audit-window-6n9U03.jsonl show zero damage events: HP/max decayed60.7/60.7 through60.0/60.0, then refreshed CookedDeerMeat raised max66.4 between supervisor polls. Food remaining rose from about603 to1197seconds. Cleanup succeeded; final64HP at[11.11,31.71,27.38]. No gathering proof earned.

Implemented accepted injury-event sequence in session/view/panel without a wire change, including listener-independent tracking. Gather and VM supervisors require unchanged known injury evidence, increased maximum and a refreshed food timer to tolerate this transition; real or regenerated hits, lost evidence and unexplained losses still stop. Legacy status retains conservative HP fallback. Native harvest keeps its actual-damage latch, compares consecutive food/health samples and checks the80% floor throughout. Cleanup now stops/deletes this trial's behavior or active goal before diagnostic image transfer, preventing continued automation during a slow camera fetch.

Focused Go checks passed .checks/20261030T210040Z-tDAgV3.log; VM gather/harvest/resupply/combat checks passed .checks/20261030T210045Z-{vi7Sp6,1W8AfD,bzPKFQ,3KQxfh}.log (see individual saved check logs for exact generated names). Native HarvestTests passed .checks/20261030T210145Z-xTN19W.log and native build .checks/20261030T210148Z-hofUeq.log. Latest cleanup-order combat/goal checks passed .checks/20261030T210331Z-e1aWJJ.log and .checks/20261030T210331Z-T0y9K0.log. Full CI, committed review, landing/deployment and one changed-build live retry follow. Broad M2 acceptance remains open.

**agent:codex/renki-milestones** at 2026-10-30T21:17:16Z

PR108 merged as7700a93. Full just check ci passed (.checks/20261030T210517Z-IvoHqY.log), committed just check review passed (.checks/20261030T210710Z-uB3SYK.log), and just check vm install verified controller/plugin checksums (.checks/20261030T210745Z-YszhTF.log).

The one changed-build live trial (.checks/20261030T210819Z-kcHKdj.log; .checks/gather-trial-20261030T210820019368Z/summary.json) reached tree1:61548 after successful move_to t-1, then harvest t-2 timed out after180seconds without strike contact or harvesting. Exact audit .checks/audit-window-1FNOLW.jsonl shows three approach commitments, with physical contact Y40.98/41.87/40.14 above supported ground Y31.15/31.02/30.87. Zero damage events; final57.9HP at[21.8,30.87,35.06]; no cleanup errors, combat off, game stopped and panel online. Original Wood22/BeechSeeds17/Feathers2 remain; there was no spare food at start. Camera .checks/camera-20261030T211121661091Z.jpg and final camera .checks/camera-20261030T211235274500Z.jpg retain visual evidence. This run did not encounter another food refresh, so it does not live-prove that branch; its recorded food-refresh/real-injury regressions pass.

Added a resource-pivot horizontal physical ray before the collider-centre fallback, rejecting unreachable physical contact from the projected support before movement. Added compact probe/pivot/feet diagnostics and the recorded canopy-height reach regression; just check plugin test-core HarvestTests passed (.checks/20261030T211632Z-ZFH6pV.log). CI, committed review, landing/deployment and a single changed-build retry follow. All full delivery/repeated hauling acceptance remains open.

**agent:codex/renki-milestones** at 2026-10-30T21:23:37Z

PR109 merged the pivot-ray/contact-support reach fix as3ccd07d. Full just check ci passed (.checks/20261030T211644Z-Q6mwg8.log), committed just check review passed (.checks/20261030T211825Z-o33xFx.log), and checksum-verified just check vm install passed (.checks/20261030T211859Z-XE8tMZ.log).

The next bounded gather trial selected beech1:61671 at[45.74,33.83,35.34], initially24.17m away and clear. Native move_to t-1 succeeded, but fresh perception during approach found a Greyling30.12m from that tree, inside the unchanged32m falling protection. Controller gathering correctly refused before harvest with 'no nearby isolated tier-zero full beech at the site'. No cuts or contact-ray test occurred; do not count the pivot-ray fix as live-proven. Summary .checks/gather-trial-20261030T211928271209Z/summary.json; exact audit .checks/audit-window-WM1Kpz.jsonl; trial log .checks/20261030T211928Z-J9To8D.log. Zero damage, minimum/final57.6HP, final[44.92,33.59,34.16], no cleanup errors. Tas was logged out, game stopped, combat restored off and panel online. Starting Wood22/BeechSeeds17/Feathers2 were not harvested or deposited.

Do not rerun protected gathering unchanged or ask the owner to clear it. The new local Greyling sighting justifies one autonomous M3 acquisition/kiting/melee trial on this build, using existing health/terrain/character/retreat guards. M2 full delivery, live pivot-contact proof, remembered/chat site selection and repeated hauling stay open.

**agent:codex/renki-milestones** at 2026-10-30T23:37:30Z

M2 preparation exposed two supervisor limits. The long native return t-1 succeeded from[67.35,35.6,35.14] to[-36.94,36.09,8.03], within3m of home, with zero damage. Its supervisor used the earlier status at[-43.55,35.45,10.92] fetched before the terminal history and incorrectly aborted 'native return home did not complete: None'. Evidence .checks/resupply-trial-20261030T232916874513Z/summary.json, console .checks/20261030T232916Z-HBiENa.log. Do not claim this as food supply success.

A new at-home run checked four distinct chests, received exact empty-food snapshots and correctly aborted without transferring anything. Eleven home chests were visible; four requests cannot establish that food is unavailable. Evidence .checks/resupply-trial-20261030T233237899868Z/summary.json, console .checks/20261030T233237Z-aLt8hc.log, zero damage, final48.7/48.7HP, cleanup successful.

Implemented post-receipt scoped/health-checked refresh with at most5seconds for the completed task to clear. A different body task or stale completion refuses. Raised the finite supply search from4 to12 requests for this11-chest home, preserving the7-minute deadline,525m travel bound, injury protection, two food types and at most4 units per verified withdrawal. Reports retain distinct checked chest count and request budget. Focused just check vm test-resupply-food passes the real completion-order race, newer task refusal, permanent stale-state bound, seven-chest positive search and twelve-request cap (.checks/20261030T233606Z-mv62ur.log). A changed-supervisor supply run is now checking home under these bounds; full-tree/haul criteria remain open.

**agent:codex/renki-milestones** at 2026-10-30T23:41:10Z

The broader live supply run passed: .checks/resupply-trial-20261030T233626047846Z/summary.json, console .checks/20261030T233626Z-1NKTSz.log. Exact native receipts verified CookedDeerMeat1 and NeckTailGrilled4; automatic eating recovered Tas to84.6/84.6HP, with zero damage events and successful cleanup. This is supply evidence, not a tree-to-storage pass.

The user requested bounding searches by unique chests in the area instead of an arbitrary lower count. This supersedes the previous note's twelve-request implementation paragraph: each run now fixes its initially observed eligible chest set, checks likely stock first, stops after two verified food types, and never repeats a chest/food pair. Current contents still override memory; probing and withdrawing different observed food can require separate requests to one chest. The finite set/ordinary-food allowlist plus existing time/travel/injury/ownership guards bound work. Reports distinguish initial unique search chests, checked chests and requests. Focused tests pass14-chest success beyond the old count,14-chest exhaustion, completion-order/new-order/stale-state cases (.checks/20261030T233955Z-dKttdI.log). Full CI reruns for this user-directed default change. Tas is now fed; one full controller gather/storage/return trial is running on the already deployed native harvesting build, independently of the pending scouting fix.

**agent:codex/renki-milestones** at 2026-10-30T23:42:15Z

The first freshly fed full gather trial started from home and found zero currently perceived full beech trees. It correctly refused before submitting any native work: .checks/gather-trial-20261030T234022610220Z/summary.json, console .checks/20261030T234022Z-hBCgUQ.log. Zero damage events, final84.3/84.3HP and successful cleanup. This is an eligibility refusal, not a harvesting pass or evidence that no trees exist outside current perception. Do not repeat from the same view. After the changed-build native combat scouting trial, inspect its fresh view for an eligible M2 site. General remembered-site discovery and multi-trip work remain open.

**agent:codex/renki-milestones** at 2026-10-30T23:58:30Z

Outdoor controller gather trial on33be3ba selected isolated full beech1:61548 at[23.62,31.37,33.53] from five observed trees. Native move_to succeeded. Harvest t-2 proved pivot-ray trunk contact at[24.09,32.17,33.98] and resource broke after7 local hits: the full beech was actually felled. Before safe completion, an enemy approached within15m and harvesting refused; controller retreat followed, with1 damage event and final77.1HP at[25.9,32.33,19.45]. Evidence .checks/gather-trial-20261030T235528819650Z/summary.json, .checks/audit-window-L4JQb3.jsonl, .checks/20261030T235528Z-xBSA4E.log. Cleanup succeeded. This is live full-tree/contact proof, not log splitting, collected gains, chest delivery or full M2 completion. Do not repeat a full-tree benchmark against this now-felled tree; any log follow-up needs fresh identity/guards. One newly observed enemy supports the next autonomous M3 trial after the checked-scout-route fix.

**agent:codex/renki-milestones** at 2026-10-31T00:36:12Z

The full-beech contact/felling proof from .checks/gather-trial-20261030T235528819650Z/summary.json stands: beech 1:61548 broke after seven local hits before an enemy interrupted the remaining harvest sequence. This does not establish log splitting, collected gains, gain-only chest receipts, return or repeated hauling.

The later M3 scouting fixes are merged/deployed through 8969d8d. Its final trial completed 225.9 m of local search and return with zero damage, but final fresh perception again contained zero full beech trees. Selection was evaluated from the retained fresh observation without starting another unchanged gathering trial. Do not treat this as proof there are no trees beyond perception, and do not retry the now-felled tree.

Next M2 work is remembered resource/worksite discovery and refreshing a candidate site before beginning the strict benchmark's baseline and exact tree selection. Ordinary trees are currently excluded by notableResource in internal/memory/landmarks.go; StartWood can reuse its saved chat-created last site but otherwise relies on fresh nearby trees. Old coordinates are destination hints only, never permission to cut or proof of new logs. Preserve full tree identity, new-log and exact delivery receipts in any extended test. Full gather and multi-trip acceptance remains open. Returning this unfinished ticket to ready and releasing the claim for handoff.

**agent:codex/renki-milestones** at 2026-10-31T05:42:11Z

### Two-tree controller cycle implemented

Added maxTrees (legacy default one, bounded to three), complete per-tree new-log chains, one retained initial product baseline, and delivery before selecting another tree under capacity/axe pressure or insufficient remaining time. Native cuts remain bounded to five per tree; total work is eight minutes per requested tree, with a bounded between-tree stamina rest. Missing/unsafe descendants and native failures cannot advance to another full tree. Opt-in returnHome pins known home, uses a 120-second dry native move before storage, requires a subsequent floor-correct pose, stows through existing task-matched receipts using dry approaches, and finishes at the pinned home. Existing return-to-site definitions remain compatible and mutually exclusive with returnHome.

The owner wood command now requests up to two trees and a home finish. Bounded landmark memory records ordinary full beech hints; recent distant hints can select travel destinations, with fresh acquisition/safety checks after arrival. VM gather benchmark accepts a bounded tree count and optional known work-site coordinates, and requires the requested distinct full trees, completed log chains, exact combined gains-only deposits and final destination proof. Two-tree, pressure delivery, stale/under-floor home receipt, changed home pin, remembered-hint scope/age, evidence-substitution and baseline-preservation checks pass. Full gate/review/deployment and live proof follow; no multi-tree game completion is claimed yet.

First live destination hint is the previously observed south stand around [-43,36,-85], within 150 m of home, rather than manual enemy/tree staging. Fresh native observations must still authorize each cut. Door PR128 landed as b704d73; isolated staircase-to-home return on d2ee29b took 5.9 seconds through the current two gates, verified exact food withdrawals and indoor floor, and took zero damage. This is not proof of the directly-under-house approach.

**agent:codex/renki-milestones** at 2026-10-31T05:50:19Z

### Necessary repair prerequisite

The last verified native pack has AxeFlint durability38/100, insufficient for a dependable two-tree chain. Added explicit interact repair:true for crafting stations, behind the existing movement/AllowGather/access gates. The native routine approaches on checked dry ground, requires normal interaction sight/reach and observed current station/UI, and invokes the game's repair-button method. Game shelter, recipe, station type/level and repair eligibility remain authoritative. The installed CraftingStation.Interact API returns false even when it opens; completion therefore uses observed station/UI, not that return flag. No-cost mode is refused. A click needs one verified durability increase, emitted with task-attributed slot/prefab/before/after/max; at most32 clicks/10 seconds after opening. Completion/cancellation closes the station UI. Other station-specific worn gear is not claimed repaired.

Return-home gathering automatically performs that preparation at a currently observed home workbench when no carried axe is at least90% repaired, then waits for fresh actual inventory before leaving. Stale, failed or ineffective repair cannot advance. Cross-language protocol fixtures, parser conflict/type checks, station-kind and permission checks, native installed-game build and focused controller readiness/repair regressions pass. One fixture omitted required round-trip envelope defaults and was corrected after reading its diagnostic. Full gate/review/deployment and the live two-tree check follow. ec47825's earlier full gate/review passed; draft PR129 is linked.
