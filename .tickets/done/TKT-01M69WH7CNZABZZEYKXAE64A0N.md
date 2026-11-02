---
schema: 4
id: TKT-01M69WH7CNZABZZEYKXAE64A0N
title: Run bounded food work while idle near typed markers
type: task
status: done
status_reason: null
priority: normal
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
claim: null
archive: null
created_at: 2026-10-31T21:06:00Z
updated_at: 2026-10-31T22:32:52Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Implement the user's requested first marker-idle food slice: bounded wild-food collection, preparation of carried raw food, and surplus storage near typed markers. Use the existing behaviour scheduler and body lease, current same-world waypoint definitions, delayed idle triggers, cooldowns and explicit opt-in. Preserve meals and interrupt on owner commands, unsafe observations, injuries or marker edits. Farming crops, hunting and pressure-driven multi-site production remain separate work.

## Acceptance criteria

- [x] Marker idle triggers require fresh same-world floor-aware dwell, have cooldowns, and invalidate queued work on leave/edit/reconnect or commands
- [x] Berry collection, carried-raw cooking and surplus storage use bounded native routines and verified receipts while retaining personal food
- [x] Panel exposes opt-in marker work and last outcomes; unattended jobs do not spin on empty sites
- [x] Focused replay checks and full CI pass; deployment and earned live results are recorded

## Implementation plan

Use the existing serialized behaviour engine for a marker-scoped idle trigger with fresh observation dwell, per-run cooldown and finite run budget. Pin marker identity/version for queued and active work; discard on world/session changes, leaving the area or marker edits. Preserve command takeover through existing body leases.

Implement a marker_food Go template that chooses bounded edible pickables at berries markers, native cooking of carried raw ingredients at cooking markers, or matched surplus deposits to currently perceived local chests at storage markers. Reuse collection, storage and cooking receipt verification; keep personal meals and reject threats, injury, water, stale observations and capacity issues. Show opt-in enable/disable and outcomes in the marker editor. No hunting, crop cultivation, automatic fire fueling or cross-site production loop in this first slice.

Add replay tests for scheduling, cancellation, marker changes, empty sites, reserves and receipts. Run focused checks, just check ci, commit and just check review before push. Deploy the controller preserving game state, run bounded supervised live food/marker checks, and record earned evidence plus remaining limits.

## Notes

**agent:codex/renki-milestones** at 2026-10-31T21:17:01Z

First marker food slice implemented on t3/marker-idle-food (stacked above t3/food-acquisition). Typed marker idle trigger uses 30-second dwell, floor-aware current waypoint pin, owner-command leases, five-minute cooldown and 12 completed attempts per controller session. Marker food runner collects up to four wild-food targets, cooks up to three carried raw ingredients with matched native delivery traces, or stores surplus in currently perceived in-reach boxes while keeping two meals per kind. Panel provides opt-in controls and outcomes. New reusable just vm marker-food-check supervises health, keeps full diagnostics in .checks, and yields to owner takeover.

Focused Go checks passed (.checks/20261031T211104Z-pUBmDQ.log); desktop/mobile marker UI checks passed (.checks/20261031T211335Z-isGK1q.log). First full just check ci passed (.checks/20261031T211342Z-CwCm6k.log); final gate rerun follows small review fixes. Tas replenished through existing food goal d56ffffc32aced16a5641519, safe home and online (.checks/food-20261031T210551Z). New runner live collection/cooking/deposit checks still pending; no crop cultivation or cross-marker production loop claimed.

**agent:codex/renki-milestones** at 2026-10-31T21:33:53Z

Live results so far: controller-only deployment preserved the client. The new idle trigger waited before dispatching native cooking at home. Initial trial .checks/marker-food-20261031T212031Z exposed PowerShell array envelopes in the supervisor (fixed/tested). Trial .checks/marker-food-20261031T212125Z exposed a fresh perception arriving 60 ms ahead of state; strict future-game-time rejection cancelled the batch, and one inserted RawMeat subsequently burnt. The guard now checks absolute state/perception skew and wall-clock freshness, with a regression test. No injury occurred.

Trial .checks/marker-food-20261031T212422Z cooked RawMeat at station 1:70084, observed normal removal and spawned CookedMeat 647877508:2930, but native collection did not finish in its bound. Full trace window: .checks/audit-window-cc30tV.jsonl. A later ordinary pickup t-1 retrieved the exact output (.checks/task-5f9v0kif.json), and Tas ate/healed normally. The drop settled ~0.85 m from its original spawn goal, below the pickup task's old 1 m replan threshold. Added a replay of those coordinates and native pickup tracking for 0.25 m movement/current-item arrival. This needs a DLL deployment and positive integrated cooking retry. Separately reserve background meal upkeep during active marker cooking, matching the existing durable food goal's receipt protection.

The original berries marker was attempted with normal dry/hostile-avoiding movement; native routing refused the route. The bounded helper returned Tas to safe home, online and injury-free (.checks/marker-food-20261031T213009Z). Added south mushrooms from three remembered sources as another explicit food marker; live visit pending. Existing home cooking marker points at working indoor cookers; original automatic cooking marker still points to unheated outdoor stations. Station 1:70085 retains burnt food; ordinary generic interaction correctly refuses cooker use, so automated station cleanup remains unsupported. Do not label natural foraging or integrated new-marker cooking successful from the primitive pickup alone.

**agent:codex/renki-milestones** at 2026-10-31T21:45:05Z

Native pickup fix deployed and checksum-verified (.checks/20261031T213609Z-alAGmk.log). The idle cooking trial .checks/marker-food-20261031T213736Z completed its behaviour with runs=1,last=completed. Native t-3 succeeded: cooked one RawMeat and collected one CookedMeat, followed by normal eating. Full audit: .checks/audit-window-dMiwS0.jsonl. The supervisor then unnecessarily commanded a home movement while already sheltered nearby, causing about two points of fire damage. Overall trial is correctly failed; no claim of clean whole-trial success. Skip that redundant return now. New meals may increase max HP before healing, so admission remains 80% health and continued work accepts at least 40%/25 HP while independently stopping for actual injury. Controller-only fix deployed (.checks/20261031T214038Z-gaNbA1.log); full just check ci passed (.checks/20261031T214039Z-PcBRGR.log).

The southern mushroom route was also refused (.checks/marker-food-20261031T213322Z); native traces identify nearby Greyling/Boar/Neck exclusion corridors. Do not relax safety or repeat these eastern/southern routes blindly. A western mushroom hint near [-71,33.5,-71] is the next guarded alternative and still needs fresh confirmation. Natural food acquisition remains unproven.

Clean cooking retry .checks/marker-food-20261031T214114Z yielded at preflight because a new live follow task t-3 took the body. No food work or logout occurred in that retry. Audit .checks/audit-window-LW9C3E.jsonl confirms nearby player chat and follow movement. Leave this command intact while awaiting the user's preference; RawMeat1 remains carried. Lasting idle presets remain disabled pending clean live validation. PR: https://github.com/warricksothr/renki/pull/134.

**agent:codex/renki-milestones** at 2026-10-31T22:12:17Z

User released Tas from following for the remaining food trials. Fresh state is idle, sheltered at home with 25/25 HP, no active meals and RawMeat1. Production resupply .checks/food-20261031T220959Z boundedly searched 11 unique chests and stopped needs_plan because the Mushroom chest 1:70040 was in use and cooking requires an active meal. Another chest was unreachable, not empty. Cleanup logged Tas out and stopped the client safely.

The accessible raw-food chest reported MushroomYellow7, but ready-food selection excluded this valid edible. Extend production resupply, marker surplus storage and food-check staging to recognize yellow mushrooms; preserve native CanEat and health gates. Replay the exact failed-probe-to-verified-starter flow and reserve-preserving yellow-mushroom deposit. Focused just check controller test-packages ./internal/goal ./internal/behaviour passed (.checks/20261031T221151Z-OtUAM2.log). Deploy the controller after full CI, then run resupply, clean idle cooking, western forage and storage in that order. No new natural gathering or whole-trial success earned yet.

**agent:codex/renki-milestones** at 2026-10-31T22:24:17Z

Yellow-mushroom support passed full just check ci (.checks/20261031T221216Z-YhLhXU.log), committed review (.checks/20261031T221531Z-sffeTE.log), and controller deployment (.checks/20261031T221453Z-Twu7LD.log), pushed as 25af8cc. Resupply .checks/food-20261031T221507Z succeeded with an actual mushroom withdrawal/eating/recovery in two tasks; the previously busy Mushroom chest had become free. This does not live-prove yellow-mushroom withdrawal, which is replay-tested.

Clean cooking attempt .checks/marker-food-20261031T221631Z still failed collection: station 1:70085 spawned CookedMeat802026012:103, but its dry pickup route stalled. Cooking itself had no injury; Tas remained online. Audit .checks/audit-window-O0qwzK.jsonl shows dry navmesh endpoint [-33.6,36.21,10.53] within the existing 1m pickup radius, but outside the walker's default 0.5m final radius. NavmeshCandidates consequently rejected the useful close endpoint. Ordinary exact-ID pickup t-5 then recovered the output (.checks/task-dt5cwbo3.json), but standing at that pickup point caused repeated 2HP fire hits before the western visit. Audit .checks/audit-window-tRmYS0.jsonl records these; food regeneration restored health. Do not characterize the separate recovery as injury-free.

Align pickup planner and task arrival radius, allow normal sight/range-checked hand pickup to avoid walking closer to a visible item, and return completed marker cooking to its marker waiting point. Actual floor/visibility, normal inventory/ownership, dry routing, injury and receipts remain enforced. Recorded-position native replay passed (.checks/20261031T222229Z-C9nFEK.log); cooking-return regression passed (.checks/20261031T222314Z-R4Q5fy.log). Full just check ci passed (.checks/20261031T222048Z-RnGODS.log). Native deployment and changed-build cooking retry are next, after the current western visit returns.

Western idle gathering .checks/marker-food-20261031T222103Z earned runs=1,last=completed with Mushroom2 -> Mushroom4 and unchanged injury baseline2187. Home return is still in progress at this note; do not yet claim a complete round trip or storage delivery.

**agent:codex/renki-milestones** at 2026-10-31T22:32:51Z

Native kitchen changes deployed and checksum-verified as916d4e4 (.checks/20261031T222803Z-u0U1rO.log). The changed-build idle trial .checks/marker-food-20261031T222941Z succeeded completely and retained Tas online in safe shelter. Native t-3 cooked one NeckTail and collected one NeckTailGrilled; t-4 returned to the marker waiting point; t-5 ate the product normally. Injury baseline remained0 for this connection, including the subsequent recovery observation. Full audit .checks/audit-window-ZDEqPX.jsonl. Final verified state85.2/85.2HP, sheltered at[-34.07,36.09,8.93], idle.

Western forage .checks/marker-food-20261031T222103Z completed its local idle batch with Mushroom2 -> Mushroom4 and unchanged injury2187. Initial home return t-12 timed out; bounded recovery t-14 ultimately returned to verified shelter without injury. The overall supervisor remains failed for the first timeout. Filed draft TKT-01M6A1D13A2A9WXTEZH9MS1WZA (Avoid repeating stalled stair approaches on raised-home return) under M2; do not mark that routing defect fixed.

Surplus storage .checks/marker-food-20261031T222655Z succeeded through actual idle dwell and matched native deposit receipts: Mushroom3 -> Mushroom2. This used naturally collected surplus, after ordinary meal refresh, not a staged chest withdrawal/deposit. Test cleanup disabled all finite trial definitions.

Enabled and verified the canonical local presets for home cooking, home storage and west mushrooms through the authenticated idle-food API. They require30s idle near their own current marker/floor, yield to body commands and danger, and retain five-minute cooldown/session attempt bounds. There is no automatic trip between these sites, crop cultivation, hunting or ingredient withdrawal in these marker presets. Last full just check ci: .checks/20261031T222048Z-RnGODS.log; extra cooking-return replay: .checks/20261031T222314Z-R4Q5fy.log; committed just check review: .checks/20261031T222424Z-OZDtAU.log, no findings. PR134 remains draft on the existing stack; broader M2/M3 and production goals remain open.

## Summary

Implemented, committed, pushed and deployed marker-scoped idle food work in PR134 (https://github.com/warricksothr/renki/pull/134). Local wild-food gathering, carried-ingredient cooking and reserve-preserving storage have earned live evidence. Enabled the three validated presets; current Tas is safely sheltered and fed. Native pickup tracks settling products, aligns route/pickup radius, uses ordinary visible hand reach and returns marker cooking to its waiting point. Automated checks and committed review passed. The western home-return timeout remains a separate draft navigation bug; this does not close broader M2/M3 or cross-site food production work.
