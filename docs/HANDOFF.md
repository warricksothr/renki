# Handoff

Replace this page at each handoff; tickets retain decisions and evidence.
Handoff of 2026-11-17 from `agent:claude-code/chests`.

Start a fresh worktree with `just setup-worktree`, then `just vm doctor`.
Read AGENTS.md, docs/VM.md ("Start here"), docs/ROUTES.md and docs/MEMORY.md.

## State

Tas (Tasleroch) is online at saved home [-204.39, 32.97, -467.43] on the build
of branch `t3/route-memory`, fed, with cooked deer meat and sausages carried.
Private video is running. Loot and automatic logout remain disabled. The VM
lease is released. The game had been offline from 2026-11-04 to 2026-11-17,
130 game days; Tailscale on the VM had signed out and is signed in again.

PRs #179 to #182 are merged. Open: the route memory recording slice.

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
- Route memory, slice 1: recording and `just vm routes` (docs/ROUTES.md).

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
   ground Tas has walked), in progress, claimed by this session's actor. Slice
   2 is the planner with `just vm route-plan`; slice 3 walks plans, marks
   failed links, and is used by the census and the home survey.
2. TKT-01M62VXZ84SA9TSJC6NJNPV8KJ (Controller: assign every item to chests,
   with a loose-items chest): the storage plan, waiting on route memory. The
   owner wants one item type per chest for bulk goods, related chests grouped
   together, and shared chests for low-volume categories such as trophies.
   The game's item type is too coarse to group by: 56 items are "Material".
   The last census is `.checks/census/20261117T192626Z.json`: 52 containers,
   153 distinct items, 15 items split across containers.
3. TKT-01M62VXYWJHB5BC9VGCBM0PVXD (Remember what each chest holds): claimed;
   looks built, needs its criteria confirmed against the tests.

## Known gaps

- The home survey still cannot leave the upper floor; route walking fixes it.
- TKT-01M7N9VE23Q3HJPXN5CWRR1ZKR (Inspect: no approach found when Tas starts
  flush against a chest row) is a draft. The census works round it.
- The census's walk toward a container is an ordinary move with no scope.
- `TestSurveyEngineStartStopAndOwnerTakeover` in `internal/app` is timing
  sensitive and failed one gate run; it passed alone and on rerun.
- The GitHub runner once lacked the dotnet CLI; rerunning the job passed.
- Older items from the 2026-11-04 handoff still stand: the 32 old in-progress
  claims, unearned live criteria for portal following, Loot reach and bedtime
  midnight delivery, and the saved `goal/<hash>` instances that warn as
  unknown. Do not promote drafts or take old claims without the owner.
- Never print `vm.local`, tokens, keys or passwords. Do not kill `renki-ctl`.
