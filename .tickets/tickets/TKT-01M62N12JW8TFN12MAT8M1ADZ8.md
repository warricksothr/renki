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
  branch: t3/gather-site-ready
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: db85ea9744ea663b4137578c1090ca45b9cd8587
  session: null
  claimed_at: 2026-10-30T17:46:20Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-30T21:17:16Z
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

PR108 landed food-refresh/injury evidence as7700a93; CI, committed review and checksum-verified installation passed. The changed-build gather attempt reached beech1:61548 but timed out in native approach with no physical strike/hit: measured contacts were40.14–41.87m high while feet/support were about31m. Audit .checks/audit-window-1FNOLW.jsonl and summary .checks/gather-trial-20261030T210820019368Z/summary.json show zero damage, final57.9HP and successful cleanup. Native horizontal probes currently aim through each collider AABB centre; a full-tree canopy can shift that centre away from the narrow trunk slice and force a high centre-ray fallback. Approach then walks repeatedly toward unreachable ground below that contact.

Add a pivot-centred physical horizontal ray before the existing collider-centre/fallback probes, preserving Collider.Raycast as contact evidence. Check physical contact reach from the proposed supported standing position before issuing movement, returning a concise refusal rather than spending180seconds underneath an unreachable point. Include probe method/pivot/feet in compact approach diagnostics. Verify a canopy-offset narrow-trunk regression, high contact refusal and existing tilted-log/raised-floor/occlusion policies, then focused checks, full CI and committed review. Land/deploy and run one changed-build trial; record exact outcome without rerunning unchanged or owner staging. Full delivery, repeated hauling and positive M3 combat remain open.

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
