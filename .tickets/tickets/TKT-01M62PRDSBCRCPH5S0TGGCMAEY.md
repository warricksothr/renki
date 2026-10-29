---
schema: 4
id: TKT-01M62PRDSBCRCPH5S0TGGCMAEY
title: Claim a bed near home, or tell the owner there is none
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - plugin
  - chat
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62PRDRKEHTQ1WC12FQ3S11J
  - TKT-01M62BMD18K3SE1WXJ7FDN8EPT
  - TKT-01M62EP0PDAB02DNJAJ6WS3MYD
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-resume
  branch: t3/stow-home-approach
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: b325d9dc1d00f1613f4c4fe678239b486339983f
  session: null
  claimed_at: 2026-10-29T22:16:44Z
  expires_at: null
archive: null
created_at: 2026-10-29T02:10:24Z
updated_at: 2026-10-29T22:41:17Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

Nimah, 2026-10-28: the bot should claim a bed close to home if one is available, and if not let the owner know. A claimed bed is also where the game respawns it: today it wakes at the world spawn after every death, and walks home from there.

### What to do

- **Perception (6.2):** beds also report whether someone owns them (`owned`), beside the existing `mine`, so the controller can tell a free bed from one that is taken.
- **The controller looks for a free bed** within a short distance of home (from perception when there, from memory otherwise). It claims the nearest by walking to it and interacting, which sets the bot's spawn point.
  - **When:** on `set home`, when it has no bed of its own on joining a world, and after a respawn far from home.
  - **When the game refuses** (no roof, too wet, a ward), it moves on to the next bed or reports the reason.
- **No free bed:** it tells an owner once, plainly, for example "There's no free bed near home. If you build me one, I'll claim it." It does not nag; it says it again only when asked (status), or when home moves.
- **Status** says whether the bot has a bed and where.

## Acceptance criteria

- [x] Perception says whether a bed is owned, and 6.2 and the fixtures say so
- [ ] With a free bed near home, the bot claims it and respawns there after a death
- [ ] With none, it tells an owner once and says so in status
- [ ] A refusal from the game (no roof, a ward) is reported with its reason

## Implementation plan

Start with the owner's explicit sleep/claim commands. Add nullable bed ownership to live perception (unknown must not mean free), select the nearest suitable owned bed or an explicitly unclaimed nearby bed, and recognize sleep/go to sleep. Add an optional bed-only sleep flag to interact so one body task can claim/set its spawn point, wait briefly for the ownership RPC, then use the bed again to lie down. Keep all game exposure/fire/wet/enemy checks and relay refusals. Verify ownership after claiming and do not report sleep merely because the spawn point changed. Owner stop/death/reconnect cancels the task through the existing task runner, with no delayed continuation.

Test controller selection, unknown/taken beds, command ownership and refusals, wire fixtures/validation and bounded asynchronous bed-use policy. Inspect the current beds before choosing live checks; do not reset or take someone else's bed or create a destructive death check. Run just check ci and committed just check review before landing/deploying. Automatic claiming on home/join/respawn, persistent bed status and a positive respawn-at-bed check remain a subsequent slice of this ticket; leave those acceptance criteria open.

## Notes

**agent:codex/renki-resume** at 2026-10-29T22:29:16Z

Manual sleep/claim slice implemented: sleep/go to sleep/go to bed chooses the bot's own visible bed, otherwise a confirmed unclaimed nearby bed. Nullable owned metadata prevents inferring free ownership from old/unknown data. interact sleep:true claims/sets spawn, waits at most three seconds for ownership, then uses the bed again. Only actual attachment to that bed establishes in-bed success. Game checks/refusal messages are retained, including night/fire/roof/wet/enemy conditions. State distinguishes in_bed from actual sleeping. Unknown/taken selection, aliases, owner gating, incompatible args, shared wire fixtures and asynchronous ownership are covered by the gate (175 C# tests plus controller checks).

Live at 22:22:51–22:22:52Z: sleep selected explicitly free bed 1:38548, set spawn, confirmed ownership, and attempted the second use. The game refused "You can't sleep at this time" (day 51, fraction about 0.34). Perception then showed mine:true,owned:true; the other bed 1:38536 remained another player's. Evidence .checks/audit-window-6vVaDr.jsonl. The claim succeeded, but the broad PlayerBase effect-area trigger wrongly counted as interaction reach, so Tas had not approached it.

Fixed interaction reach to ignore trigger colliders and trigger ray hits. After full deployment, go to sleep at 22:28:17–22:28:19Z actually walked Tas from (-36.53,36.09,7.96) to (-32.03,36.09,11.97), close enough to the physical bed collider, and relayed only "You can't sleep at this time" as a refused task. HP25 throughout. Evidence .checks/audit-window-GuhIsv.jsonl. Sent him back to configured home afterward. No player's bed was reset and no death/time-of-day was staged.

Only acceptance criterion 1 is complete. Criterion 2 includes actual respawn at the bed, which has not been tested. Automatic claiming on home/join/respawn, persistent bed status and automatic no-free-bed notices remain unimplemented; nighttime positive sleep and roof/fire/wet/enemy/ward staging remain live checks. Owner stop/death/reconnect uses the existing task cancellation; there is no scheduled continuation after cancellation. Do not call the entire bed ticket or M2 complete.

**agent:codex/renki-resume** at 2026-10-29T22:41:17Z

Nighttime manual sleep passed on deployed 3232dad at 22:40Z–22:41Z: Tas approached his claimed bed, interact succeeded with actual bed attachment, and current state reported in_bed at (-31.08,36.55,15.22), HP25. He was waiting for the other player, so actual shared time-skip sleeping is not claimed. Evidence .checks/audit-window-RxI1SV.jsonl. Sent him home after recording this to continue the approved route-memory deployment/checks. Actual respawn-at-bed, automatic claiming triggers and persistent bed status remain open.
