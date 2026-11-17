# Handoff

Replace this page at each handoff; tickets retain decisions and evidence.
Handoff of 2026-11-17 from `agent:claude-code/chests`.

Start a fresh worktree with `just setup-worktree`, then `just vm doctor`.
Read AGENTS.md, docs/VM.md ("Start here"), docs/ROUTES.md and docs/MEMORY.md.

## State

Tas (Tasleroch) is online in the warehouse near [-155.6, 31.0, -471.5], not at
home, on the build of `main` at PR #189 plus branch `t3/walker-gather`, with
sausages carried.
Private video is running. Loot and automatic logout remain disabled. The VM
lease is released. The game had been offline from 2026-11-04 to 2026-11-17,
130 game days; Tailscale on the VM had signed out and is signed in again.

PRs #179 to #189 are merged.

## Owner decisions, 2026-11-17

- Route memory is the top priority and blocks other work.
- Tas may use ordinary multi-leg moves on runs he starts himself.
- Navigation debug output belongs in the regular recipes and APIs, so an
  agent can read it and guide the bot.
- Deploy and merge as work completes.
- Work round game limits in the controller where needed.

## What was built

- `just vm doctor`, `just vm up`, `just setup-worktree` (docs/VM.md).
- Container census: the `census` behaviour, `just vm chest-census`, and
  `Tas, take inventory` in chat (docs/BEHAVIOURS.md, "Container census").
  Gone containers are forgotten from memory and landmarks. The chat command
  has not been tried in game.
- Route memory (docs/ROUTES.md): recording and `just vm routes`; the planner
  and `just vm route-plan`; and walking a plan, used by the census. On
  2026-11-17 a census started on home's upper floor crossed to the warehouse
  by itself, and a later one opened 45 containers across both buildings.
  Memory holds 134 walked cells, none failed.
- A built piece is never hostile. A training dummy near home had stopped
  every run within 20 m and closed the bounded walker's routes for 40 m.
- A timed-out inspection no longer ends a census. It is charged 2 m of travel
  for each second it was given. The owner has not been asked about this.

## Why route memory

The bounded dry walker gave up three ways where an ordinary `move_to` got
through: leaving the upper floor by the stairs (both home surveys failed with
no travel), going round a double chest row, and crossing between the
warehouse and the east building. Proven legs, used by hand this session:
home, stair top [-193.01, 32.97, -471.11], stair foot [-189.51, 31.26,
-471.72], warehouse west [-169.69, 31.03, -467.8]; barrel aisle
[-163.35, 30.97, -467.12].

## Next

1. TKT-01M7NEG203A7HHNMQE0F8PW3AM (Route memory: plan building routes over
   ground Tas has walked), in progress, claimed by this session's actor. The
   owner agreed this order on 2026-11-17, and docs/ROUTES.md, "Walking a
   plan", says what each piece does:
   - Done: the generic walker (`Navigator`, `MoveStyle`, `NextLeg`); the
     `travel` behaviour behind chat's `go to` and the panel's go button; the
     `detour` fallback in stow, collect, gather, marker food, marker cargo
     and the day rest.
   - Left, step 2: go home (`internal/home/home.go`), the corpse run
     (`internal/chat/brain.go`) and retreat. They dispatch one owner task
     and track it, so they need more than a call to `StartTravel`.
   - Left, step 4: the food goals and return home in `internal/goal`. They
     save each pending leg; `NextLeg` is the form made for them.
   - Left, step 5: the home survey. It stops on any leg without a `survey:`
     receipt and checks every point against the base boundary, so decide
     how an ordinary leg is charged and bounded before writing it.
   Live so far: the census crossing between buildings and the panel's go.
   No `detour` fallback has run live, nor chat's `go to`, nor the budget
   fixes from the review of PR #185.
2. TKT-01M62VXZ84SA9TSJC6NJNPV8KJ (Controller: assign every item to chests,
   with a loose-items chest): the storage plan, waiting on route memory. The
   owner wants one item type per chest for bulk goods, related chests grouped
   together, and shared chests for low-volume categories such as trophies.
   The game's item type is too coarse to group by: 56 items are "Material".
   The fullest census is `.checks/census/20261117T192626Z.json`: 52
   containers, 153 distinct items, 15 split. The newest,
   `20261117T214213Z.json`, opened 45 and left 20 unchecked: four inspections
   timed out while a Greydwarf Elite stood about 37 m from the warehouse.
3. TKT-01M62VXYWJHB5BC9VGCBM0PVXD (Remember what each chest holds): claimed;
   looks built, needs its criteria confirmed against the tests.

## Known gaps

- The home survey still cannot leave the upper floor (see Next, 1).
- `just check ci` failed twice on 2026-11-17 on one panel UI scenario with no
  checks run (`mobile/inspection-peel`, then `mobile/shared-regression`) and
  passed on an unchanged rerun. The cause is not known.
- TKT-01M7N9VE23Q3HJPXN5CWRR1ZKR (Inspect: no approach found when Tas starts
  flush against a chest row) is a draft. The census works round it.
- The census's walks are ordinary moves with no scope.
- `TestSurveyEngineStartStopAndOwnerTakeover` in `internal/app` is timing
  sensitive and failed one gate run; it passed alone and on rerun.
- The GitHub runner once lacked the dotnet CLI; rerunning the job passed.
- Older items from the 2026-11-04 handoff still stand: the 32 old in-progress
  claims, unearned live criteria for portal following, Loot reach and bedtime
  midnight delivery, and the saved `goal/<hash>` instances that warn as
  unknown. Do not promote drafts or take old claims without the owner.
- Never print `vm.local`, tokens, keys or passwords. Do not kill `renki-ctl`.
