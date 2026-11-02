---
schema: 4
id: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
title: "Behaviours: templates and instances the bot runs on its own"
type: epic
status: draft
status_reason: null
priority: high
due_on: "2026-12-06"
labels:
  - controller
  - chat
assignees: []
milestone: M5
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-11-02T23:06:27Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-lead
  name: ""
extensions: {}
---

## Description

The bot's routines are built from a few templates rather than coded one by
one. A **template** is a built-in kind of work, written in Go, that says what
parameters it takes and how it runs: `loot`, `stow`, `fuel`, `gather`. An
**instance** is a template with its parameters filled in, plus a name, a
trigger, conditions, how it repeats, a priority, and whether it is on.
Instances are kept per bot.

Owners clone an instance and change its parameters ("stow wood" becomes
"stow stone" by changing one field), and start, stop, switch on and switch
off any of them from chat, the panel or, later, the agent.

Organising the base comes down to one template: `stow` moves all of some
items to an ordered list of chests, and says when they do not fit, so
players know more chests are needed or something must be cleared out.
Grouping items into chests on its own comes later (the base upkeep epic).

### Decisions so far (2026-10-28)

- Behaviours and their switches are per bot, not per world.
- Loot pickup is on by default, and leaves an owner's deliberate drops alone.
- Stowing uses chests inside the base unless an instance names others.
- Survival and direct orders always come before every instance.

## Acceptance criteria

- [ ] Loot pickup and deliver-when-full run as instances of templates, not as hand-written routines
- [ ] An owner can clone an instance, change a parameter and start it from chat or the panel
- [ ] When items do not fit, the owners are told what and how much, and the panel shows it

## Notes

**agent:claude-code/renki-review** at 2026-11-02T06:11:09Z

Roadmap (docs/ROADMAP.md, 2026-11-01): only criterion 1 (loot and deliver as template instances) gates M2, and it is largely met by the gather, collect and stow templates. Criteria 2 (clone and edit from chat or the panel) and 3 (storage notices in the panel) are scheduled with M5. Proposal for the owner: drop M2's dependency on this epic, or split criterion 1 into its own ticket, so M2 can close on its own criteria.

**agent:claude-code/renki-plan** at 2026-11-02T16:20:55Z

Applied on 2026-11-02 by the planning pass: milestone set to M5, and the M2 epic no longer depends on this epic. Its owner-editing outcome is M5 goals 5.4 and 5.5 (docs/ROADMAP.md), and its only M2 part, loot and deliver as template instances, is the same work as TKT-01M62N12HJ (Controller: loot and deliver-when-full behaviours for hauling). Children stay in the milestones whose goals they serve. The owner can revert this.

**agent:claude-code/renki-lead** at 2026-11-02T23:06:27Z

Decision, owner, 2026-11-02: idle and passive behaviours.

This supersedes "direct orders always come before every instance" for one case. An owner's follow may be lent to **passive** instances. Every other direct order still comes first, and survival always does.

- **Idle** instances run only when Tas has nothing to do, as before.
- **Passive** instances run alongside an owner's follow:
  - they borrow the body only while the followed player is settled (the settle tracker from TKT-01M6EYEMRV);
  - they work inside that player's settle area;
  - they give the body back when the player leaves the area or gives any order, and the follow restarts.
  - A passive instance never starts a follow itself.
- **When Tas goes passive** it runs a **passive list**: an ordered list of parameterised instances. There is a built-in list; the agent side (internal/voice) may supply its own, which is validated before it runs, with a fallback to the built-in list when it is absent, invalid or stale. The agent revises the list on events (stash counts, food reserves, what an owner says), never on every tick. A list starts as data, instances of existing templates; Starlark procedure packages (docs/PROCEDURES.md) come later, if lists prove to need logic. The agent cannot grant permissions, widen the area or override orders.
- **Needs** re-rank work inside the settle area; they do not send Tas away. A trip to fill a need is idle or goal work.
- **Defaults**: loot and work-alongside gathering are both passive and on by default.

Tickets:
- TKT-01M6EWN4WH: passive mode in the engine.
- TKT-01M6F877Q8WJ2YPMG4JMY9HYCV: the anchor's activity.
- TKT-01M6F877TXKB11SZQ1D7JW4DF0: working alongside.
- TKT-01M6F877XVE8DPB4WA9WXQVJKE: the needs ledger.
- TKT-01M6F8780VQAZ938F3MY56GS8V: the agent-supplied passive list.
