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
updated_at: 2026-10-30T20:22:23Z
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

Validate the existing controller gather template with one bounded autonomous full-tree -> new logs -> observed product gains -> exact home-chest deposit receipts -> return-site cycle. Add just vm gather-trial using a uniquely named manual once-only behavior; native tasks own movement and work. Require stable world/connection, supplied axe/capacity/food, isolated tree and remembered home storage, and abort on health loss, nearby hostiles, scope changes or bounds. Stop/delete the temporary behavior even after a lost response, then logout/stop and keep the offline panel available. Verify exact behavior-attributed requests, distinct resource IDs/local-hit breaks, preservation of original products, measured gains/receipts and final return. Cover forged/incomplete proof and supervisor failure/cleanup paths with focused checks, then run just check ci and committed just check review before landing. Diagnose the first live outcome using compact retained evidence; do not repeat unchanged failures or ask for owner staging. Site inference/chat, repeated hauling and broad M2 acceptance remain open.

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
