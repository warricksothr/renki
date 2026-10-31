---
schema: 4
id: TKT-01M69R5A75H3CTVWY175RZ9PS1
title: Acquire missing food through bounded foraging and hunting
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - safety
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
  branch: t3/food-acquisition
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: aa39fe27533640f972b4ad549a88de71235da80c
  session: null
  claimed_at: 2026-10-31T19:50:40Z
  expires_at: null
archive: null
created_at: 2026-10-31T19:49:36Z
updated_at: 2026-10-31T20:42:44Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Extend production food resupply after storage exhaustion with bounded normal resource collection, remembered foraging sites, explicit eligible wildlife hunting and verified home return. Preserve native permissions, current body ownership, injury guards, task receipts and durable attempt budgets. No manual supply staging is required.

## Acceptance criteria

- [x] Food and fuel resources use exact yield, fresh pack and scoped result evidence; no success-text-only gains
- [x] Foraging and hunting attempts and visits persist with finite limits, fresh eligibility and normal combat permissions
- [x] Food acquired outside home returns on the original floor before cooking, completion or escort handoff
- [ ] Focused replay checks and complete CI pass; guarded live results and remaining cases are recorded

## Implementation plan

Extend new food goals with a typed, optional acquisition checkpoint; old saved goals keep their admitted storage-only behavior. Search storage first, then exact live ready-food/raw drops and pickables with validated yields. Save resource and site attempts before dispatch (12 resources, six remembered food sites, at most two explicitly named ordinary wild Boar/Neck hunts). Resolve pickable products only by fresh measured inventory gain or a fresh matching nearby spawned drop; no animation/result text counts as inventory. A kill needs the exact native task, target and observed species death; it is independent from acquiring ingredients.

Remembered berry/mushroom/branch positions are hints within 80 m of pinned home, never stock or route evidence. Native dry/hostile-avoiding travel owns routing; a failed site consumes one visit and does not assert empty coverage. Current native permissions and combat guards remain authoritative. Hunting requires current attack permission, active food, stamina and eligible isolated wildlife. Keep existing strict injury stop; never reset injury to admit a continuation.

After every acquired yield, return to the original home level before cooking, finishing readiness or handing back to escort. Attempts, post-interact waits and uncertainty survive restart without replay. Keep summaries compact. Add replay checks for source spoofing/missing yields, partial pickup, products and auto-pickup, scoped home arrival, persisted exhausted visits, unsafe wildlife/current permissions and checkpoint failure. Run focused just checks, full just check ci, commit/review/push a PR stacked on food-resupply, then deploy the controller and supervise real production food goals. Record unavailable supplies or cold-fire cases honestly; no manual staging requests or fabricated items.

## Notes

**agent:codex/renki-milestones** at 2026-10-31T19:59:42Z

Implemented an optional typed acquisition checkpoint on new food goals; saved older food goals retain their original storage-only scope. After storage exhaustion, normal exact-yield drop/pickable collection can supply ready food, raw ingredients or fuel wood. Product RPC/auto-pickup handling requires fresh measured pack gain bounded by the observed yield, not success text. Short product waits, resource IDs, visits and two-hunt attempts are persisted before dispatch with the existing native task budgets and uncertain-action rules.

Remembered edible/branch sites within 80 m are travel hints only; six visits, consolidated in four-metre cells, do not assert area coverage. Twelve resource actions are allowed. Current native attack permission can admit only exact fresh isolated unstarred wild Boar/Neck targets; normal native equipment, ground, bystander and retreat checks still decide. Target-lost/unreachable hunts can advance to distinct prey, never retry the same ID. An exact task/target/species observed death is distinct from obtaining any ingredient. Existing injury baseline remains strict. Every collected yield outside home returns to the original floor before cooking/completion/escort continuation; a below-floor arrival cannot pass.

Added compact panel acquisition counts and --require-acquisition to just vm food-check. Corrected controller-only deployment to preserve an offline client while bringing back the independent panel. Focused just check controller test-packages ./internal/goal ./internal/app ./internal/chat passed in .checks/20261031T195547Z-S9C5dC.log; subsequent checks include the added distinct-prey negative recovery. Initial full gate caught stale panel output; rebuilt through just check panel build. Final just check ci is running in .checks/20261031T195717Z-pOcFBT.log. Live acquisition, combined raw-to-cooked fallback, cold-fire fueling and escort continuation are not yet claimed. Tas remains reserved for automated food trials; no manual setup requests.

**agent:codex/renki-milestones** at 2026-10-31T20:08:16Z

Full CI for the supervisor follow-up passed140s (.checks/20261031T200408Z-JG7Wdr.log). The initial live run .checks/food-20261031T200147Z verified a normal one-tail deposit then encountered a supervisor sampling race after six storage tasks: status named a completed goal task while the newer checkpoint already named the next. Fixed local supervision to accept only exact completed task IDs in this goal's bounded history (with final native outcomes), refresh both samples once for the dispatch/checkpoint window, and still reject an owner's new task or unproven task. Focused just check vm test-food-check passed.

The second live run .checks/food-20261031T200408Z earned real acquisition: goal c395aa1d54278d059e3bb0b4 normally picked up NeckTail drop2402772000:167 with task t-9 and measured inventory gain1, then returned to the pinned home floor through task t-10. No injury was observed. Cooking t-11 at station1:70085 failed unreachable before inserting any food. It tried48 relative-angle seeds:114 floor candidates,73 body rejections,41 reach/fire-footing rejections, zero walk attempts. This is the same station used successfully in .checks/cooking-20261031T182424Z from another arrival bearing. The earlier success used position[-33.28,36.22,10.74]. Native approach sampling is being extended with fixed world bearings independent of arrival angle, preserving one seed per tick, all support/body/sight/fire checks, four walks and the60s deadline. No cooking success is claimed by the new goal yet. Tas was safely logged out and the independent panel restored after the failure.

**agent:codex/renki-milestones** at 2026-10-31T20:18:00Z

Native fixed-bearing cooking sampling passed just check plugin test-core CookingApproachesTests and full just check ci136s (.checks/20261031T200844Z-GB47Y9.log); committed/reviewed/pushed0a41550 with no review findings (.checks/20261031T201130Z-Ylr0jM.log). just check vm install verified controller and plugin package/installed checksums (.checks/20261031T201155Z-BNcuy9.log).

Live goal56244f17e16a09ae75ac81ce in .checks/food-20261031T201244Z successfully used the fixed approach at station1:70085 and cooked the previously naturally collected NeckTail. Native taskt-9 earned the full matched removal/output/pickup/inventory receipt, outputNeckTailGrilled1, remote_correlation. No actual cold-fire fueling occurred. The goal then unnecessarily visited a mushroom site because readiness away from the home marker fell through into acquisition. It did earn live foraging at mushroom1:86252 (taskt-11, fresh measured gain1) and eating that new Mushroom (taskt-12, observed active-food refresh).

The extra slot increased max health from53.4 to68.4 while HP stayed53.4; injurySeq stayed0. A nearby threat then correctly stopped the goal before return. Cleanup logged Tas out at the eastern forage site near[11.42,33.22,-8.2] and restored the panel. This is not full goal success or keep-online proof. Do not mistake the food maximum increase for an injury. Native startup recovery remains enabled and may refuse an unsafe outdoor rejoin; no manual enemy-clear request is needed.

Fixed host ordering so an already satisfied reserve/nourishment/health predicate dispatches its home return immediately before any extra source search. Added a regression with a tempting extra live berry source and cooking outside the home marker; it must return and complete in exactly two tasks, preserving cooking proof. Focused just check controller test-packages ./internal/goal passed (.checks/20261031T201538Z-IG1iTm.log); full just check ci is running in .checks/20261031T201606Z-0avurn.log. Wider M2/M3, cold-fire fueling, live hunting and escort continuation remain open.

**agent:codex/renki-milestones** at 2026-10-31T20:28:26Z

Return-order fix committed as178337a after full just check ci148s (.checks/20261031T201606Z-0avurn.log), clean just check review aa39fe2..HEAD (.checks/20261031T202135Z-uofXx9.log) and pushed. just check vm deploy-controller passed (.checks/20261031T202154Z-6v0Bsu.log); offline panel reports178337a and the game remained offline.

The saved outdoor login exposes a separate recovery gap: passive StartupRecovery correctly refuses exposed healing and all body tasks, so it cannot bring an adequately supplied body home. Implementing an explicit expiring same-world home return inside native recovery, without disabling the guard or lowering its ordinary healing release. The one-shot order comes from the exact world's saved controller home, is consumed before movement, and requires50HP/70%max, two foods,25stamina, dry grounded admission and6m hostile clearance. Normal native dry/hostile-aware movement stays under120s/160m travelled/80m from home; injury, close threats, water, lost permissions/session, stop or low stamina log out. Home floor/shelter/12m clearance hands back to passive95% healing. All ordinary body tasks remain blocked. The reusable just vm recover-home WORLD supervisor records bounded observations/audit and cleans up on failure; no manual supply/enemy-clear requests.

Focused just check plugin test-core RecoveryReturnTests passed (.checks/20261031T202431Z-dx1Lpj.log). Initial plugin build caught a V3/double-array adapter mismatch before deployment; corrected and just check plugin build passed (.checks/20261031T202626Z-uDQvNl.log). just check vm test-recover-home passed (.checks/20261031T202728Z-wPHY0U.log). New recovery is not deployed or live-proved yet; Tas remains offline at the last recorded eastern forage site. Full food goal, cold-fire fuel, live hunting and escort handoff remain unearned.

**agent:codex/renki-milestones** at 2026-10-31T20:35:05Z

Recovery implementation cea8d61 passed full just check ci136s (.checks/20261031T202826Z-6VtK5c.log), clean just check review aa39fe2..HEAD (.checks/20261031T203051Z-8lc5h7.log), and package/installed native/controller checksums (.checks/20261031T203104Z-9sm2Kk.log). First helper attempt stopped before game launch on a PowerShell reserved variable; corrected to savedHome. The arm preflight then found Windows SSH drops an empty quoted WorldBase64 argument on clear; the recipe now omits that optional parameter. Actual arm/clear passed on BLIGHT. Follow-up full just check ci140s passed (.checks/20261031T203213Z-BhWUxn.log).

just check vm recover-home DizzysHellheim passed (.checks/20261031T203218Z-6Yollp.log). Evidence .checks/recovery-20261031T203218Z and audit .checks/audit-window-iAKBxb.jsonl retain native startup return from[11.43,33.22,-8.2] to observed original home floor/shelter at[-37.54,36.09,8.87]. Native traces show return start20:33:02Z, arrival20:34:06Z and passive healing release.349 in-world state frames have minimumHP53.4; final66.8/66.8 and controller injurySeq0. Sampled travel83.4m includes the checked approach around/below the raised house before reaching the proper floor; no below-floor success was accepted. The one-shot order was consumed/cleared and Tas remained online, idle, sheltered with three active foods. No guard was disabled, items fabricated or enemies manually cleared.

Next just check vm food-check --reserve2 --keep-online also passed (.checks/20261031T203429Z-0QLxxv.log). Recording the exact goal/storage/native receipts next; this is whole production food readiness success, not proof of a new combined acquisition/cooking sequence or cold-fire fueling. Wider M2/M3, cold-fire fuel, live hunt and escort handoff remain separate open checks.

**agent:codex/renki-milestones** at 2026-10-31T20:36:38Z

Exact follow-up production food goal6c40aee16794f82b1e5573bb succeeded in two native steps with reserve2 and keep-online verified (.checks/food-20261031T203429Z). Taskt-2 normally withdrew3 Blueberries from1:70031 with matched transfer/fresh-pack receipt; no cooking was necessary. Final66.2/68.5HP, same sheltered home-floor position, no injury reported. Tas stayed logged in between trials.

That fresh chest receipt now contains actual supported raw stock (NeckTail3 and RawMeat4), unlike the earlier empty-stock snapshots. A normal one-tail withdrawal passed through just check vm task-check withdraw (.checks/20261031T203613Z-qzG5sb.log). Beginning a new guarded production goal using that carried ingredient and existing normal stow-ready staging into excluded chest1:70040; this should earn complete cook/readiness/return evidence without pretending food acquisition or cold-fire fuel was exercised in this run. The real raw-drop pickup and live mushroom foraging remain independently earned in the earlier trials.

**agent:codex/renki-milestones** at 2026-10-31T20:42:44Z

Complete carried-raw cooking/readiness/return benchmark now passed: just check vm food-check --reserve1 --storage <eight bounded chests> --stow-ready1:70040 --require-cooking --keep-online, evidence .checks/food-20261031T203634Z and .checks/20261031T203634Z-GfNlWF.log. Goal49a97e3b768d462d64b37fdc succeeded in10 native steps. Normal staging deposited2 Blueberries into excluded1:70040; carried raw NeckTail came from the separately verified normal one-tail withdrawal. Taskt-15 at station1:70085 earned full direct_spawn cooking removal/output/pickup/inventory proof forNeckTailGrilled1. Taskt-16 returned to the original home floor; final70/70HP, position[-34.86,36.09,7.36], safe keep-online verified. This directly exercises the return-before-extra-acquisition fix. Actual cold-fire fuel was not exercised. Earlier natural raw pickup/home return and mushroom foraging/eating stay separate primitive evidence; no new full acquisition-to-readiness goal is claimed.

Added bounded food-to-escort verification to just vm food-check --reserve2 --escort Nimah --keep-online. It requires the persisted continuation run/task identity, matching native request plus accepted/running receipts, ten seconds of quiet same-leader follow/wait at safe home, then a verified stop; collection/defense remain disabled for this handoff-only check. Focused supervisor and compact-progress checks passed; full just check ci142s passed (.checks/20261031T203926Z-oJ3xPR.log). Fixed compact trial status for integer food task counts/string goal IDs and selecting only timestamped evidence directories, avoiding fixture directories and full-log context dumps. No live handoff was attempted.

The user emergency-logged Tas out after a separate base attack while code checks were running. Audit .checks/audit-window-SNFFE5.jsonl shows Greydwarf damage7.97 at20:39:43Z, normal retreatt-20 accepted/running within0.3s, and Greydwarf_Elite damage27.73 at20:39:58Z. Flee made very little movement inside the house. Last native in-world observation20:40:00Z was43.7/67.1HP at[-33.13,36.09,5.99]; no death was observed. just check vm stop and panel-start both passed (.checks/20261031T204033Z-DDrmkJ.log and .checks/20261031T204040Z-Ie9bSf.log). Tas is offline, the independent panel is online, RecoveryGuard remains enabled, and no rejoin will be attempted into the known attack. Keep live escort handoff, cold-fire fuel, hunting and wider M2/M3 open; trapped native escape is a distinct safety failure to prioritize.
