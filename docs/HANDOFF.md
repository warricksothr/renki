# Handoff

Replace this page at each handoff; tickets retain decisions and evidence.
Handoff of 2026-11-18 from `agent:claude-code/chests`.

Everything this page relies on is on `main`, on a ticket, or on the VM.
Nothing lives only in a worktree: no unmerged branch, no ticket claim and no
saved evidence file is needed to carry on.

## Start here

1. Make a worktree from `origin/main` and run `just setup-worktree`, then
   `just vm doctor`. Doctor is read-only and names the fix for each gap.
2. Read AGENTS.md, docs/VM.md ("Start here"), docs/ROUTES.md and
   docs/MEMORY.md.
3. Pick an actor name, `agent:tool/session`, and use it on every ticket
   write and for the VM lease.
4. Claim the ticket you take up. This session released its claims.

Never print `vm.local`, `plugin/game.local.props`, tokens, keys or
passwords. Do not kill `renki-ctl`. Do not promote drafts or take old
in-progress claims without the owner. Keep Loot and automatic logout off.

## State on 2026-11-18

- Tas (Tasleroch) is online at home, [-202.4, 33.0, -467.6], in the world
  DizzysHellheim. Private video is running. The VM lease is free.
- `main` is at the merge of PR #191. PRs #179 to #191 are merged and nothing
  from this session is open.
- The VM runs the code of that merge. Doctor names its build as branch
  `t3/walker-home`, the last branch deployed; `main` differs from it only in
  ticket notes and this page.
- Route memory on the VM holds 161 walked cells and no failed links
  (`just vm routes`). It covers home's upper floor, the stairs, the yard and
  the warehouse.
- Container memory on the VM holds 76 containers
  (`just vm panel GET /api/memory/containers`). The census reports saved
  under `.checks/census/` were local to one worktree and are not needed:
  memory on the VM is the record, and `just vm chest-census` refreshes it.

## Owner decisions

Made on 2026-11-17 and still standing:

- Route memory is the top priority and blocks the storage work.
- Tas may use ordinary multi-leg moves on runs he starts himself.
- Navigation debug output belongs in the regular recipes and APIs, so an
  agent can read it and guide the bot.
- Deploy and merge as work completes, without asking each time.
- Work round game limits in the controller where needed.
- The walker is converted in this order: the generic walker; a shared trip
  for chat, the panel and going home; the base-bound behaviours; the food
  goals and return home; the home survey last.
- For storage: one item type per chest for bulk goods, related chests
  grouped together, and shared chests for low-volume kinds such as trophies.

Not yet put to the owner, and theirs to reverse:

- A timed-out inspection no longer ends a census. It refuses that container
  and charges 2 m of travel for each second it was given.
- A fallback walk is ordinary moves: it does not avoid hostiles or water as
  the native move does.
- Retreat was left unconverted on purpose. It is an escape, and a planned
  walk over remembered ground may not suit it.

## What exists now

All of it is documented where named; this is the map.

- **Setup**: `just setup-worktree`, `just vm doctor`, `just vm up`
  (docs/VM.md).
- **Container census**: the `census` behaviour, `just vm chest-census`, and
  `Tas, take inventory` in chat (docs/BEHAVIOURS.md, "Container census").
  Containers proved gone are forgotten.
- **Route memory** (docs/ROUTES.md):
  - recording, and `just vm routes` to read it;
  - the planner, and `just vm route-plan TARGET [FROM]`, which prints the
    legs or the reason there is no route;
  - the walker (`internal/behaviour/route_walk.go`): one ordinary move per
    leg, arrival checked on the intended floor, a ground failure marking
    its link, the way planned again at most twice. It plans through a
    `Navigator`, makes moves in a `MoveStyle`, and has a stateless form,
    `NextLeg`, that nothing uses yet.
- **Where the walker is used**:
  - the census, for every walk it makes;
  - the `travel` behaviour, behind `Tas, go to <place>`, `Tas, go home` and
    the panel's go and home buttons, when a way of two legs or more is
    known;
  - the `detour` fallback, tried once when a native move finds no way: in
    stow (so also collect, gather, marker food and marker cargo deposits),
    marker cargo's return, collect's and gather's moves to and from their
    site, gather's walk home, marker food's returns and the day rest's
    bedside move.
- **Plugin**: a built piece is never hostile, in perception and in native
  route checks. A training dummy near home had stopped every run within
  20 m and closed the bounded walker's routes for 40 m.

## What ran in the game, and what did not

Ran live on the VM:

- A census started on home's upper floor crossed to the warehouse by
  itself, and a later one opened 45 containers across both buildings.
- The panel's go button took Tas to the warehouse and to a bed upstairs.
- The panel's home button brought him from the warehouse up the stairs.

Unit tests only:

- Every `detour` fallback. Each needs its native move to fail in the game,
  and none was staged.
- Chat's `go to`, `go home` and `take inventory`.
- Fixes made after review: the census budget accounting, and never walking
  on over a cancelled leg. Several had no further review round; each pull
  request says which.

## Next

1. **TKT-01M7NEG203A7HHNMQE0F8PW3AM** (Route memory: plan building routes
   over ground Tas has walked), ready and unclaimed. Its notes carry
   the history. What is left, in the agreed order:
   - The trips home Tas starts himself (`Keeper.GoHome` in
     `internal/home/home.go`: when idle, or after a failed follow or move)
     and the corpse run (`internal/chat/brain.go`). Each dispatches one task
     and follows it by id, so the tracking needs reworking; a call to
     `StartTravel` is not enough.
   - The food goals and return home in `internal/goal`. They save each
     pending leg. `NextLeg` was written for them.
   - The home survey (`internal/goal/survey.go`). It stops on any leg
     without a `survey:` receipt and checks every point against the base
     boundary. Decide how an ordinary leg is charged and bounded before
     writing it. Until then the survey cannot leave home's upper floor.
   - Four of its five criteria are ticked. The one left, grouping
     containers and stations by building and floor, belongs with the
     storage plan below.
2. **TKT-01M62VXZ84SA9TSJC6NJNPV8KJ** (Controller: assign every item to
   chests, with a loose-items chest), ready and unclaimed: the storage plan.
   Route memory no longer blocks a first version. Wanted: container roles
   (one item, a shared category, loose items, do not touch), neighbouring
   containers treated as a group, and our own category table, because the
   game's item type is too coarse (56 items are "Material"). Propose a
   layout from container memory for the owner to approve before anything
   moves. The fullest census, on 2026-11-17, saw 52 containers and 153
   distinct items, 15 of them split across containers.
3. **TKT-01M62VXYWJHB5BC9VGCBM0PVXD** (Remember what each chest holds, its
   size and when it was seen), ready and unclaimed. It looks built. Confirm
   its criteria against the tests and close it.
4. The owner also asked for help setting up OBS for streaming development.
   Another session was working on it; nothing was done here.

## Known gaps

- `just check ci` is flaky in two places, both unrelated to this work and
  both passing on an unchanged rerun. One panel UI scenario at a time ends
  with no checks run (`mobile/inspection-peel`, `mobile/shared-regression`).
  `TestSurveyEngineStartStopAndOwnerTakeover` in `internal/app` is timing
  sensitive. Neither cause is known.
- A census can end with containers unchecked when hostiles stand near the
  base: each inspection they block times out at 45 s. On 2026-11-17 four
  did, with a Greydwarf Elite about 37 m from the warehouse, and 20
  containers were left unchecked.
- TKT-01M7N9VE23Q3HJPXN5CWRR1ZKR (Inspect: no approach found when Tas starts
  flush against a chest row) is a draft. The census works round it.
- Walks made by the census, `travel` and `detour` are ordinary moves with no
  scope: they are not held inside a radius.
- The GitHub runner once lacked the dotnet CLI; rerunning the job passed.
- If the panel stops answering, check Tailscale on the VM first. It had
  signed out while the game was offline from 2026-11-04 to 2026-11-17.
- Older items from the 2026-11-04 handoff still stand: the old in-progress
  claims of other sessions, unearned live criteria for portal following,
  Loot reach and bedtime midnight delivery, and the saved `goal/<hash>`
  instances that warn as unknown.
