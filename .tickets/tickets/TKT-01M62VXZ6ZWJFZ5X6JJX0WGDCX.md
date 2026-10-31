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
updated_at: 2026-10-31T18:03:48Z
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

Continue the explicitly authorized multiplayer cooking work on PR131. Preserve normal UseItem/Interact and owner-mediated RPCs; never claim network authority. Replace the local-simulation admission restriction with two explicit delivery evidence modes. Locally executed SpawnItem remains direct evidence. For remote removal, witness the single normal RPC's requested yield/owning peer and correlate only newly observed item IDs absent from the pre-removal snapshot, created by that peer in a short spawn window and close to the computed output origin, with the expected prefab/one-item stack and recorded crafter when the station provides it. Observe the synchronized slot becoming empty and actual pickup/inventory gains for the accepted IDs. Document that remote correlation is not a direct spawn witness and refuse ambiguity, extra output, old drops, merges, lost/missing output and unrelated inventory gains.

Add pure receipt policy tests for old/reloaded/unrelated/late drops, delayed bonus output, duplicate identities, excess candidates and pickup/slot ordering. Capture exact requested yield through a narrowly gated observation of the normal outgoing removal RPC, so asynchronous bonus delivery cannot complete on a partial count. Keep all safety/access/capacity and single-consumption/single-removal rules. Update the native proof tool to accept explicit direct or remote evidence without dropping pickup/inventory checks. Run focused checks then just check ci, commit/review/push, deploy while offline and repeat a guarded real batch. Keep full evidence in .checks and bounded diagnostics in tool output. If a new physical reach/fire prerequisite appears, diagnose and improve the ordinary native approach instead of bypassing it. Wider food resupply/escort composition and fermenter/beehive/ore acceptance remain open until proved.

## Notes

**agent:codex/renki-milestones** at 2026-10-31T16:51:23Z

Implemented the first native cooking slice on t3/cooking-food, stacked above PR130. Owners/operators can request cook food; fresh carried raw boar/deer meat or neck tail and an empty heated station within12m are required. Native interact cook uses normal item/station use and pickup, current recipe/access/safety/capacity gates, one owned input-slot receipt and one removal attempt. Scoped normal SpawnItem and Humanoid.Pickup hooks tie spawned instances and actual pickup inventory gains to this batch. Uncertain insertion/removal is never repeated; loss, foreign slots, burning or interruption fail without fabricated success. Generic pickup merge substitution is disabled for this provenance-sensitive child. Native food consumption of the output waits until its receipt completes.

Added current fire fuel/lit and cooking slot/heat telemetry, shared fixtures, protocol/policy/permission/cancellation/command tests, docs/COOKING.md, and just vm cooking-check. The live-check recipe can join under the enabled native startup guard, inspect unique current home chests for one raw ingredient using verified receipts, and always logs out/stops after admission while preserving panel/video. It prints compact results and retains full evidence. Ordinary food resupply candidate defaults are unchanged; the helper now optionally supports an explicit ingredient allowlist. Ghost-only memories cannot inflate the cooking search set.

Focused cooking core checks, controller protocol/chat checks, native build, cooking supervisor and existing resupply supervisor checks passed. Full just check ci is running. No live cooking success yet. This is a single explicit batch primitive and proof tool, not the complete automatic tending/resupply/escort-maintenance composition. Fermenter/beehive/ore AC stay open, as does live proof of avoiding burnt food.

**agent:codex/renki-milestones** at 2026-10-31T17:05:06Z

The first guarded live proof on fd2ac62 safely withdrew one RawMeat from chest 1:70031 with matched native container/inventory receipts. The supervisor then falsely timed out because it matched task response id rather than the protocol's re field. Raw failed evidence is retained in .checks/cooking-20261031T165430Z; no cooking was attempted. Corrected the parser and added a regression against the shared native task_failed fixture.

Both current nearby cooking stations (1:70085 and 1:70084) were empty but cold; their supporting campfire 1:70049 had zero fuel. Added native preparation using one carried Wood through normal Fireplace.UseItem. It matches actual station fire-check points against inactive burning-area geometry, requires a unique empty wood fire, ward access and sight/reach, and confirms wood consumption, synchronized fuel increase and actual station heat. Unconfirmed or blocked fires never trigger another fuel attempt. Chat and proof-tool selection admit cold empty stations only with carried wood. Broader automatic resupply, repeated tending and fermenter/beehive/ore AC remain open.

Focused cooking core, supervisor, chat and protocol checks passed. Full just check ci is in progress. Tas remains logged out inside home, about30.9/30.9 HP, with the retrieved raw meat and carried wood. Native startup recovery guard remains enabled; home.afterOutage remains temporarily false. Next: commit/review/deploy and repeat one continuously supervised batch. Raised-floor reach remains a possible live blocker; sight/access checks will remain enforced.

**agent:codex/renki-milestones** at 2026-10-31T17:09:28Z

The second guarded trial on9590dfc joined healthy, approached station1:70085 from inside home, and refused before fuel or ingredient use because the station was not locally simulated. No injury was observed; final health30.8/30.8 and feet[-33.25,36.15,9.21]. The corrected response parser recognized the terminal result and cleanup logged out/stopped Tas and restored the panel. Evidence: .checks/cooking-20261031T170704Z. The panel's video track remained live through offline standby.

Inspection of the actual normal cooking path shows it claims an unowned station during ingredient use. Adjusted native admission to allow this normal path while still refusing another peer's station before consumption and requiring local simulation after insertion for exact output-spawn provenance. The plugin itself never claims ownership. Added owner/local-peer trace fields on refusal and surfaced native refusal detail directly in the compact proof output. Full just check ci is running for this follow-up; positive cooking evidence remains pending.

**agent:codex/renki-milestones** at 2026-10-31T17:15:11Z

Cooking slice and live status: fd2ac62 adds native one-input cooking/provenance and the proof tool;9590dfc adds one-wood supporting-fire preparation plus the corrected re terminal parser;20885bd permits the normal game's acquisition of an unowned station, refuses occupied stations before fueling, and reports native failures compactly. All are pushed in draft PR131 above130/129. Latest installed package is20885bd, with verified controller/plugin checksums.

Validation completed: just check ci passed147s for9590dfc (.checks/20261031T170340Z-mAczvF.log) and139s for20885bd (.checks/20261031T170903Z-NYzTWl.log). Both follow-up commits passed just check review HEAD~1..HEAD with no findings. just check vm install passed for each deployed revision.

The final guarded trial (.checks/cooking-20261031T171213Z; outer log .checks/20261031T171213Z-jlFcb2.log) selected station1:70085 and refused before fuel/input use. New trace confirms hasOwner=true, owner772281326, localPeer1315392252: this is a remotely simulated station, not the unowned case handled by normal UseItem. Native result: the cooking station must be unowned or simulated locally to verify its output. Full raw evidence remains saved. The earlier successful one-RawMeat chest withdrawal stands; no actual food insertion/cooked output has yet been proved, and no cooking AC is checked.

Tas is logged out and the VM game stopped; controller20885bd and panel remain online. Last final-trial observed health about30.8/30.8, no actual injury observed, feet[-33.25, 36.15, 9.21] indoors near the cooking racks. Retrieved raw meat and wood were not consumed in either cooking attempt. Browser confirmed offline standby video readyState4 with a live video track after cleanup. RecoveryGuard remains true; home.afterOutage remains temporarily false to avoid unsolicited startup movement.

Next cooking work is multiplayer output provenance or selecting a station genuinely simulated locally. Do not force network ownership merely to satisfy the proof, infer success from disappearance, or weaken the exact pickup/inventory receipts. Automatic resupply/escort cooking composition, bulk tending and the broader fermenter/beehive/ore work remain incomplete. The existing cooking command is a single explicit batch and replaces the current body; it does not automatically resume escort.

**agent:codex/renki-milestones** at 2026-10-31T17:40:50Z

Implemented a separate remote-correlation delivery mode for normal multiplayer cooking; removed the local-simulation admission restriction without claiming ownership or editing packets. A narrowly gated observation of this task's ordinary outgoing removal RPC captures exact requested yield and simulating peer. New output IDs must be absent from the pre-removal snapshot, have that creator, match prefab/one-item stack, birth within the short request window, appear near the computed ordinary spawn origin, and match the crafter when the station records one. Local SpawnItem remains a separate direct witness. The native result and proof tool explicitly report direct_spawn versus remote_correlation.

Remote delivery waits for the complete requested yield, replicated empty slot, actual per-ID pickup gains, full matching inventory gain and a three-second observation window. Delayed bonus output cannot complete a partial batch. Old/reloaded/wrong-owner/wrong-crafter/out-of-window drops, extra candidates, merges, missing receipts and unrelated gains do not prove delivery. Eight seconds are allowed for output acknowledgment, without another removal attempt. The normal removal RPC must be witnessed even for direct delivery.

This supersedes the prior note's local-only proof prerequisite, not its prohibition on forced network ownership or fabricated success. Vanilla drops lack a station ID; same-owner/same-prefab concurrent drops at the same place/time can be observationally indistinguishable when no crafter is recorded. docs/COOKING.md documents that limitation and conservative failure on delayed replication/early merges rather than calling remote correlation exact causality.

Focused CookingTests, native plugin build, supervisor receipt/count/identity regressions and formatting passed before the final additional request-observer check and no-crafter regression. Full just check ci is in progress and will include those additions before the native build/tests phase. No new live cooking attempt yet.

**agent:codex/renki-milestones** at 2026-10-31T17:48:53Z

Expanded focused CookingTests pass, including the final normal-request receipt and no-crafter path. Native plugin build and supervisor identity/count tests pass. Full gate initially stopped on formatting in the new bounded diagnostic trace; formatting was corrected. Two subsequent full gates reached46/46 passing panel scenarios but failed only in contact-sheet assembly: Chromium reported that mobile-controls-keyboard-waypoint.png could not be decoded despite complete=true/naturalWidth390. The retained PNG opened successfully, and the exact isolated mobile controls-keyboard/contact-sheet recipe passed. Evidence logs: .checks/20261031T174216Z-U0jRF7.log, .checks/20261031T174440Z-QUE0p8.log, and .checks/20261031T174728Z-uCUBgc.log.

Corrected the incidental CI artifact issue in panel/ui/run.mjs: source screenshots are fetched/decoded individually into bounded thumbnails; originals remain intact and linked, and the portable HTML contact sheet embeds the thumbnails. This avoids keeping all full-resolution sources active in Chromium. The focused thumbnail contact-sheet recipe passed (.checks/20261031T174800Z-6yh1Cn.log). Full just check ci is running again on the final code. No production panel behavior changed and Tas is still offline. No live success is claimed yet.

**agent:codex/renki-milestones** at 2026-10-31T17:55:34Z

Multiplayer implementation762339a and contact-sheet correction4250c87 passed full just check ci in139s (.checks/20261031T174813Z-snOXKx.log) and just check review HEAD~2..HEAD without findings. They are pushed to PR131. Verified install succeeded in10s (.checks/20261031T175102Z-LEdWTw.log).

Guarded live trial .checks/cooking-20261031T175115Z passed the former simulation-owner admission gate, then refused the station sight/reach check before fuel or ingredient consumption. It cleaned up and left Tas offline with the panel available. The outer trial log is .checks/20261031T175115Z-jx6LU6.log. No cooked output has been attempted/proved yet.

Investigation found the task used an arbitrary2.5m eye-to-collider range while the installed game's Player.FindHoverObject uses Player.m_maxInteractDistance, default5m, for normal interaction. Corrected cooking station/fire reach to the actual configured finite positive range while retaining the existing physical obstruction check. If already actually visible/reachable, the task skips an unnecessary walk to the station pivot. This is the first correction before implementing larger candidate-standing-position search: if an actual obstruction remains, it must still fail and be solved through native approach/routing, not sight bypass. Other interaction tasks retain their existing limits; scope is cooking. Full checks and another guarded trial follow.

**agent:codex/renki-milestones** at 2026-10-31T18:03:48Z

Range correction2348218 passed full just check ci in142s (.checks/20261031T175539Z-MBs7D2.log), review without findings, and verified installation (.checks/20261031T175827Z-wf9PUA.log). Guarded retry .checks/cooking-20261031T175844Z still refused station sight/reach before consumption, so the actual obstruction remains after using normal game range. No cooked-output attempt/proof yet; Tas is offline after cleanup.

Added native CookingApproach rather than bypassing sight or driving Tas externally. It first tests actual station/fire usability. Otherwise it measures48 ring seeds (one per physics step), considers live support above/below raised floors, rejects steep/wet support, standing-body collisions, blocked station/fire sight and current/future burning areas, and passes only measured candidates to the existing dry/hostile-aware MoveToTask. Four walks at most,12s each,60s total. Actual reach is checked every step; arrival near the pivot is no longer sufficient. Rejection counts and each selected position are traced for bounded diagnosis.

Refactored UseReach so actual Near delegates to the same ray/collider check from supplied hypothetical feet; actual behavior is preserved. Fire preparation exposes its unique candidate and burning bounds for approach checks without adding fuel or claiming ownership. Physics-dependent placement is checked through native build plus guarded live trial, not fabricated unit mocks. Existing core sight, movement, cooking and receipt checks remain in full just check ci. Native build is running; full gate/review/deploy/live trial follow.
