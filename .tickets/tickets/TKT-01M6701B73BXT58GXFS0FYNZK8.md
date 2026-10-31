---
schema: 4
id: TKT-01M6701B73BXT58GXFS0FYNZK8
title: Generalize durable goals for multistep food search
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
  branch: t3/cooking-food
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: db3d986e726fec11034b106bdedd68f4e1350361
  session: null
  claimed_at: 2026-10-31T18:51:46Z
  expires_at: null
archive: null
created_at: 2026-10-30T18:09:31Z
updated_at: 2026-10-31T20:43:28Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Follow docs/PROCEDURES.md and docs/PROCEDURE-TOOLS.md. Preserve M2/M3 priority and separate their acceptance from procedure infrastructure. Generalize the existing combat-specific goal host through explicitly typed reconstructible phases, one admitted food adapter at a time. There is no generic child checkpoint protocol today; registering arbitrary behaviour.Runner values is insufficient. Keep the restricted Starlark grammar and existing permission/body ownership boundaries.

Implement a host-owned durable candidate/attempt ledger, structured expected-negative versus execution-fault outcomes, and per-native-task intents, budget charging, run-scoped identities and receipts. A wrong remembered food hint must lead to a fresh scoped observation and another allowed chest. Repeat detection uses action, target and meaningful observation revision; distinct empty boxes are not identical retries. Food success requires the declared host predicate and matched transfer plus fresh inventory/consumption evidence appropriate to that predicate.

Add compact decision/evidence diagnostics and bounded scenario capture before automatic model integration. Reconcile interrupted actions only when retained proof suffices; otherwise preserve uncertainty and existing explicit-resume rules. Existing resupply_trial.py is reference/test supervision, not an implemented production goal. Unsupported crafting, repair, exploration and arbitrary durable behavior composition remain outside this slice.

## Acceptance criteria

- [x] Explicit food adapter phases persist state and charge every native task without claiming arbitrary child-runner checkpointability; bundled combat behavior is preserved
- [x] Wrong or empty food hints update a durable scoped candidate ledger and select another allowed chest; unchanged retries stay bounded across restart
- [x] Host food completion requires declared receipt and fresh inventory/consumption proof; uncertainty, cancellation, current permissions and spent budgets cannot be bypassed
- [x] Compact diagnosis links decisions, host context, scoped native receipts and candidate outcomes and distinguishes missing evidence from zero observed events
- [ ] Replay covers partial transfers, distinct empty chests, exhausted search, uncertain restart, checkpoint failure and owner takeover; separately authorized bounded live resupply earns its native evidence

## Implementation plan

Implement food_resupply as a typed durable adapter in the existing goal host, preserving the combat path. Pin home and a finite set of unique visible home chests; persist every chest/item attempt, current phase, before-pack receipt context, native dispatch intent and charged task budget before action. Ready food wins before raw ingredients; gather wood only when cooking needs it. Cook at most three single-input batches toward two carried meals plus active food/recovery. Use native transfer receipts and fresh inventory/consumption observations. Retain bounded cooking receipts on the dispatched task so host completion checks full output identities/counts rather than success text. Existing normal game ownership/access/safety checks stay final.

Expose an owner/operator resupply-food command and existing goal API. Compose an explicitly requested escort through food preparation rather than silently replacing arbitrary tasks. Preserve owner takeover, stop, current permissions and explicit resume after interruption; no uncertain task replay. Integrate low-food escort maintenance only at a safe native boundary with a pinned leader and current commander list. Keep diagnostic summaries compact with explicit missing evidence. Test wrong/empty hints, partial transfers, limits, uncertainty/restart, checkpoint failure, permissions and owner takeover plus existing combat regressions.

Run focused checks through just check, then development deploy and guarded real trials without manual staging or supply requests. Find a genuinely cold usable fire and storage area without ready meals where available; record any staging prerequisite we cannot autonomously satisfy rather than fabricate a pass. Finish with just check ci, commit, just check review, push/update linked PR and verified install. Tas remains reserved for these trials.

## Notes

**agent:codex/renki-milestones** at 2026-10-31T19:15:15Z

Implemented food_resupply on t3/food-resupply above cooking PR131. Typed host phases pin home, freeze up to64 unique live home chests, retain expected-empty versus unavailable candidates and per-chest/item/phase attempts, and checkpoint/charge every native task before dispatch. Ready meals win; supported raw ingredients and wood feed the existing native cooking composition toward a small reserve. The host requires exact transfer/fresh inventory or observed consumption receipts, or full matching cooking removal/output/pickup/inventory evidence. Session retains at most64 cooking traces on the actual dispatched task; success text alone does not satisfy the predicate. Host run IDs and dispatch times disambiguate newly recorded task IDs across controller restarts; older development evidence lacks that new field and remains explicitly scoped by saved trial/audit time.

Added resupply food / stock up on food commands for current owners/operators, full checkpoint GET /api/goals?id=ID and compact status summaries. Food keeper yields its eating channel while a food goal is queued/running. Eligible empty food slots fill before counting the reserve, followed by active food>=300s and80% health. Native escort can opt into a quiet healthy/dry follow/wait resupply boundary; chat opts in. A bounded bridge queue pins connection/world/control revision and original leader. Verified readiness hands back under the same still-valid lease and current commander list. Handoff intent/task charging and assigned receipt ID are checkpointed; uncertainty is never reissued. New owner orders and current permission checks remain authoritative. No arbitrary child-runner checkpointing or model-driven authoring is claimed.

Replay checks passed for wrong/empty hints, distinct boxes, partial transfers, exhausted search, missing/wrong/native-count receipts, unchanged retries, uncertain restart, failed checkpoint writes, pinned home/injury/budgets, revoked commander, fresh task evidence, owner takeover at handoff and current dispatch permission. Native EscortTests passed for explicit opt-in and safe boundary priority. Full just check ci passed151s before the final handoff/run-ID additions (.checks/20261031T190758Z-5wf674.log); focused checks passed after those additions (.checks/20261031T191335Z-GfISWi.log). Final complete gate is running.

Live evidence: .checks/food-20261031T190130Z exposed normal empty-transfer reason invalid_target, fixed only when paired with the exact native container receipt. .checks/food-20261031T190349Z then correctly reconciled stale neck-tail memory to actual missing stock and stopped. .checks/food-20261031T190702Z earned production goal success: withdrew3 Blueberries from1:70040, normally ate1, recovered and retained the requested total3 ready meals; three native steps and matched receipts. Its stricter outer test correctly failed because --require-cooking was requested but no cooking was needed. Do not call this a cooking-fallback proof.

A staged shortage trial .checks/food-20261031T190904Z normally deposited the carried ready meals into excluded storage1:70031, with exact moved/fresh-pack receipts. The production goal checked the eight scoped candidate chests (one inaccessible), reconciled stale hints, withdrew one remaining cooked tail from1:70039, then exhausted available raw ingredients and stopped needs_plan after9 native tasks. There are no observed supported raw drops nearby. Relevant recent actual contents of1:70031 and1:70040 also contain no supported raw ingredient. Nearby cooking-backed fires remain lit/fueled. No combined raw-to-cooked live success or actual cold-fire fuel receipt is earned. Keep those checks open; do not invent supplies, alter network ownership or request manual setup as a prerequisite to committing the implementation.

The final ready-food trial .checks/food-20261031T191150Z refused a newly observed nearby threat before any task, without observed injury, then logged out. Tas remains reserved for cooking/resupply trials. Added just vm food-check with --keep-online: success can retain the client only at verified idle shelter near home with scoped surroundings and no nearby threats; failed/unsafe/uncertain outcomes log out. Native DLL updates still need restart; controller-only changes use just vm deploy-controller. No repetitive login is required between safe trials on the same installed plugin. Native escort handoff is replay-tested, not yet live-tested. Wider M2/M3, fuel proof and autonomous raw-food acquisition remain open.

**agent:codex/renki-milestones** at 2026-10-31T20:43:28Z

The final carried-raw cooking/readiness/home-return benchmark succeeded under production food goal49a97e3b768d462d64b37fdc (.checks/food-20261031T203634Z):10 native steps, full direct_spawn cooking receipt at1:70085 forNeckTailGrilled1, exact native returnt-16 to original home floor,70/70HP, safe keep-online. No cold-fire fuel was required. A separate ready-storage goal6c40aee16794f82b1e5573bb also passed with matched3-Blueberries withdrawal/fresh-pack evidence and reserve2. New native one-shot guarded startup return passed83.4m from the eastern forage site to actual home floor/shelter (.checks/recovery-20261031T203218Z), without relaxing passive critical-health healing or disabling RecoveryGuard.

Live escort handoff is still unearned. Added a bounded just vm food-check --reserve2 --escort Nimah --keep-online verifier, with exact persisted continuation task/native request/accepted/running evidence and a10s quiet safe-home observation followed by stop. Focused checks and full just check ci142s passed (.checks/20261031T203926Z-oJ3xPR.log); no live execution occurred before the user emergency logout during a later separate base attack. That attack exposed trapped normal fleeing inside the raised house, recorded on TKT-01M62N256 — Plugin: flee task, and retreating when health is low. Last native health43.7/67.1; game stopped, independent panel online, no rejoin into the known attack. Full natural-acquisition-to-readiness, cold-fire fuel, live hunting and escort handoff remain open; earlier actual raw-drop/forage primitive receipts are recorded on child acquisition ticketTKT-01M69R5A75H3CTVWY175RZ9PS1.
