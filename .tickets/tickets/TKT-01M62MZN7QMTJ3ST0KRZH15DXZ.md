---
schema: 4
id: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
title: "M2: the bot hauls and gathers wood with its owner"
type: epic
status: in-progress
status_reason: null
priority: high
due_on: "2026-11-08"
labels:
  - plugin
  - controller
assignees: []
milestone: M2
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-resume
  branch: t3/milestone-evidence
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 552c491e6b0ec2cac344cbb99a4e38a529ff19ef
  session: null
  claimed_at: 2026-10-30T13:45:00Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:39:24Z
updated_at: 2026-11-02T17:45:01Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

The benchmark, in Nimah's words (2026-10-28): "I could be chopping wood at a location and the bot is picking up what is dropped and running it back to base to store. Then later I can tell it to go chop wood and because it knows I was collecting there it can go there but it can also decide based on where trees are."

M2 is the bot as a working helper: it carries, stores and gathers. It builds on M1's movement and chat and on perception (TKT-01M62BMD1), and it needs memory (TKT-01M62EP0P) to know where the base, the chests and the work sites are.

### The two scenes

1. **Hauling.** The owner chops at a spot. The bot stays near them, picks up the wood, resin and seeds that drop, and when it is full (or the pile is gone and the owner is done) it carries them home, puts them in a chest, and comes back.
2. **Gathering on its own.** Told "go chop wood", the bot picks where to go: the place it saw the owner working, or else the nearest trees it knows of. It equips an axe, fells, splits and breaks logs, collects the wood, and hauls it home as in scene 1.

### Out of scope here

No-chop zones (places the owner wants left alone) come later, as do combat (M3) and crafting. The bot must still survive the trips: drowning on straight-line paths (TKT-01M62G71Q) is fixed in M2, because every haul is a trip home.

## Acceptance criteria

- [ ] While the owner chops at a spot, the bot picks up the drops near them, carries them home when full and stores them in a chest, then comes back
- [ ] Told "go chop wood", the bot goes to where it saw the owner working, or else to the nearest trees it knows, and fells, splits and collects
- [ ] The bot keeps hauling across trips without drowning or getting lost, and says when it cannot (no chest, chest full, no axe)

## Definition of done

- [ ] 2.1 just vm gather-trial passes 3 runs in a row (fell, split, collect, deposit, return); each summary.json records a duration under 15 min, no death and no swimming
- [ ] 2.3 Three of three wood requests (chat or the panel's wood action) go to the most recent owner work site within 150 m of home, else the nearest safe known trees within 150 m of home, and finish 2.1's loop; at least one run starts with a recorded work site (TKT-01M6EFHE80)
- [ ] 2.4 No axe gives a chat reason within 10 s and no further body task in one live run; no chest and chest full are covered by offline tests
- [ ] 2.5 Every M2 child is done, or has a note saying what is left and is moved to a later milestone; none is in progress without a note in the last 3 days
- [ ] 2.2 Two owner sessions of 30+ min, 3+ trips each, with the owner inside the bot's perception radius for >=90% of the session: the bot picks up >=80% of drops seen within 30 m of the owner, not counting deliberate owner drops, and delivers at 80% full or 60 s after the pile is gone (haul report, TKT-01M6DDFTDD)

## Implementation plan

Continue the owner-requested M2 completion. Native stump harvesting/collection/delivery and primitive chest transfers have positive evidence; full-tree felling/log contact and bounded gather/collect/stow templates are implemented. Complete remaining full-tree gather/store/return live proof, observed owner-work/pile inference and automatic multi-trip hauling, then connect go-chop-wood chat to site/storage selection. Keep tools/crafting outside this milestone as scoped. Close supporting tickets only on their actual criteria, including failure/stop/room/dry-routing checks. Current raised-log retry broke two logs but aborted on Greyling damage before collected Wood, so broad completion remains unearned.

## Notes

**agent:claude-code/renki-groom** at 2026-10-29T03:40:56Z

2026-10-28: hauling and gathering are now instances on the behaviour engine. The behaviours epic TKT-01M62VXYQB6 ('Behaviours: templates and instances the bot runs on its own') carries the engine, settings store, chest memory and stow. Base upkeep (TKT-01M62VXZ2FWD) and the agent's tools and events (TKT-01M62VXZA33E) are filed for after M2.

**agent:claude-code/renki-plan-m2** at 2026-11-02T15:58:00Z

### Remaining for M2 goal 2.5: goal review and plan, 2026-11-02

None of the three criteria or five goals is met yet. Every in-progress M2 child now has a "Remaining for M2 goal" note saying what is left, where to start and how to verify it. Each live child depends on the VM lease TKT-01M6EF5DTG (VM: exclusive lease for deploys and live trials), which should be the first thing promoted.

### Goals, what each depends on, and fixes

- **2.1, gather loop 3 in a row.** Owner: TKT-01M62N12JW. It also needs TKT-01M62N12F0 (harvest collects the drops), TKT-01M62G71QC (drowning, seen live) and TKT-01M62MB5YR (doors). Measurable once the trial writes a duration and fails on death or swimming; the deadline bounds a one-tree run today but no duration is recorded (noted on TKT-01M62N12JW). Achievable only if the first run on the current build passes or fails for one clear cause. It has never passed.
- **2.2, owner-haul scene.** Owner: TKT-01M62N12HJ, now depending on TKT-01M6EFHE2N (loot template), TKT-01M6EFHE4E (deliver at 80% full or when the pile is gone), TKT-01M6EFHE67 (drop piles with trend) and TKT-01M6DDFTDD (haul report). That is four offline pull requests, a deploy, then two 30-minute owner sessions. It is unlikely by 2026-11-08. Measurability gap: "dropped within 30 m of the owner" can only count drops the bot perceived, so a bot that lags behind shows a high ratio over few drops. Fix: the report also gives the share of session time the owner was inside the bot's perception radius, and the goal requires at least 90 %.
- **2.3, "go chop wood" picks a sensible site.** Owner: TKT-01M6EFHE80 (VM: wood-request trial for go chop wood and its refusals), with TKT-01M62N12G9 (owner work sites) and TKT-01M62N12JW. Not achievable as written, for three reasons. A supervised trial cannot type in-game chat. `planWood` (`internal/behaviour/wood.go`) refuses a home more than 150 m away and sites more than 150 m from home, so "within 300 m" never applies. And nothing records owner work sites yet. Fix: "3 of 3 requests, by chat or the panel's wood action, go to the most recent owner work site within 150 m of home if there is one, else the nearest safe known beech within 150 m of home, and finish 2.1's loop; at least one of the three starts with an owner work site recorded".
- **2.4, honest failures.** Owners: TKT-01M62N12JW (refusals in `planWood` and `gatherSupplied`), TKT-01M62VXYY6 and TKT-01M62N12C9 (chest full through `StorageNeed`), TKT-01M6EFHE80 (live no-axe run). Not achievable live as written. "No chest" live means removing every chest within 10 m of the owner's home, and "chest full" means filling a real chest. Fix: "one live run for no axe (the trial stores the axe, then restores it); no chest and chest full by offline tests; 'stops' means no body task is dispatched after the refusal".
- **2.5, tickets tell the truth.** Measurable as written. It needs a fresh note on every in-progress M2 ticket in the 3 days before 2026-11-08, and the moves proposed below.

### Proposed moves (owner decision)

- Out of M2, because no M2 goal needs what is left on them: TKT-01M62EP0PD (memory: missing-in-view and deaths) to M3, where goal 3.4 already names it; TKT-01M62PRDSB (claim a bed) to M3; TKT-01M62N12M6 (eat: ask for food) to M5; TKT-01M62VXYV3 (behaviour chat commands) to M5; TKT-01M6DCFSCK (repeat conditions) to M4 with its review epic.
- Close once evidence is linked: TKT-01M62BMD18 (perception), TKT-01M62PRDRK (interact), TKT-01M62N12DN (equip), TKT-01M62VXYWJ (chest memory, built in a15c5fd).

### The behaviours epic dependency

This epic depends on TKT-01M62VXYQB (Behaviours: templates and instances the bot runs on its own), a draft whose criteria 2 and 3 are M5 work, so this epic cannot close on its own criteria. Its criterion 1, loot and deliver-when-full as template instances, is the same work as TKT-01M62N12HJ's criteria 1 and 3, which this epic already covers through that child. Proposal: drop this epic's dependency on TKT-01M62VXYQB and move that epic to M5. Do not file a split ticket, because it would duplicate TKT-01M62N12HJ. Keep TKT-01M62VXYY6 (stow) in M2 under its current parent, since delivery uses it.

### Dependency links that hold M2 behind M3 work (owner decision)

TKT-01M62N12JW depends on TKT-01M62N12HJ, TKT-01M62N12G9 and TKT-01M62N12DN, and TKT-01M62N12G9 and TKT-01M62N12HJ depend on TKT-01M62EP0PD. So goal 2.1 waits in the graph on the hauling scene and on memory criteria M2 does not need. The gather loop works without them: gather stows its own load, harvest equips its own axe, and the memory parts it uses are done. Proposal: unlink TKT-01M62N12JW from those three, and TKT-01M62N12G9 and TKT-01M62N12HJ from TKT-01M62EP0PD.

**agent:claude-code/renki-plan** at 2026-11-02T16:20:55Z

Applied on 2026-11-02 by the planning pass, superseding the "proposed moves (owner decision)" wording in the earlier planning note: the dependency on TKT-01M62VXYQB (Behaviours: templates and instances) is removed and that epic is in M5. The memory, bed, eating, chat-behaviours and repeat-condition tickets have also moved, and the work-site, haul and chest-memory tickets no longer depend on the memory ticket now in M3. Each moved ticket carries a note. No open M2 ticket depends on later-milestone work.

**agent:claude-code/renki-plan** at 2026-11-02T16:35:32Z

Supersedes two lines of the earlier planning note. First, TKT-01M62EP0PD moved to M3 because it extends the danger memory that goal 3.4's tickets read; goal 3.4 does not name it (TKT-01M6EFHEK3 and TKT-01M6EFHS5E own 3.4). Second, the proposed unlink of TKT-01M62N12JW from TKT-01M62N12HJ, TKT-01M62N12G9 and TKT-01M62N12DN is now applied, so goal 2.1 no longer waits on goal 2.2.
