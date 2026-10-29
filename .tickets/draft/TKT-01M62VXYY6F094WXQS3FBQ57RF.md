---
schema: 4
id: TKT-01M62VXYY6F094WXQS3FBQ57RF
title: "Controller: stow template: move all of some items to a list of chests"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: M2
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies:
  - TKT-01M62VXYSN4RQV75HTA3BSD162
  - TKT-01M62VXYWJHB5BC9VGCBM0PVXD
  - TKT-01M62N12C9EWXCD54WMBH08T4E
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-10-29T03:40:48Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

"Move all of `items` from `from` to the chests in `to`, in order." This is
the core block of organising the base.

### Parameters

- `items`: prefabs, a group (`wood`, `ore`, `food`) or `*`
- `from`: the bot's pack for now. Chests and the ground in an area come later.
- `to`: an ordered list of chests. A resource with a lot of stock fills
  several, one after another.
- `keep`: how many of each item the bot holds back (food, a stack of wood
  for fires)
- `overflow`: `tell` (the default) or `leave`

### Running

1. Plan from chest memory: for each chest in order, top up partial stacks
   of the same item first, then use empty slots.
2. Walk to each chest, deposit, and read back what is inside. If a chest
   has less room than remembered, plan again from what is there.
3. Whatever is left is a shortfall.

### Reporting a shortfall

Tell the owners what did not fit and why: *"I need storage for 140 Stone:
stone-1 and stone-2 are full."*, or *"No chest is set for Resin; I have
30."* The panel shows a "storage needed" note until it is solved. No
nagging: the message repeats only when the shortfall changes or after a
long gap. The bot never throws anything away (Safety rule 8). When purging
is the answer, it says so and leaves the choice to the players.

## Acceptance criteria

- [ ] stow fills the listed chests in order, topping up stacks before using empty slots
- [ ] Several chests for one item fill one after another
- [ ] A chest with less room than remembered is planned again from what it really holds
- [ ] What does not fit is reported to the owners and on the panel, and the report repeats only when it changes
- [ ] Nothing is destroyed or dropped to make room
