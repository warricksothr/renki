---
schema: 4
id: TKT-01M7NEG203A7HHNMQE0F8PW3AM
title: "Route memory: plan building routes over ground Tas has walked"
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - movement
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/chests
  branch: t3/controller-census
  worktree: /home/sothr/.t3/worktrees/renki/t3-527f19de-ed082ea71ff42b98
  commit: f48dc9064bbbba7e1291c44f0e090c91f3daf4fd
  session: null
  claimed_at: 2026-11-17T19:30:59Z
  expires_at: null
archive: null
created_at: 2026-11-17T19:07:17Z
updated_at: 2026-11-17T21:55:10Z
created_by:
  id: agent:claude-code/chests
  name: ""
updated_by:
  id: agent:claude-code/chests
  name: ""
extensions: {}
---

## Description

On 2026-11-17 at the new home, the cautious bounded walker gave up three separate ways where an ordinary `move_to` got through:

- **Stairs.** Two home surveys failed their first leg from the upper floor with no travel. Ordinary moves through the stair top and stair foot recorded on 2026-11-04 reached the warehouse.
- **Round a chest row.** A barrel behind a double row failed four bounded approaches to a verified standing spot 5 m away. An ordinary move to that spot went round the row and the barrel opened.
- **Between buildings.** From the east building, six containers 12 to 15 m away in the warehouse failed with "no safe local route" and no standing spot measured.

The base has several buildings, stairs, inclines and upper levels. Each behaviour now works round this on its own (the census walks to a standing spot or toward the container). That does not scale to stowing, sorting, surveys and returns.

### Proposal: route memory in the controller

Keep what the body has actually walked, and plan over it.

- **Record proven links.** Native trails already come back in survey and inspection receipts, and state frames give the body's position. Turn completed walks into per-world links between points, each with its height, so floors stay distinct.
- **Learn connectors.** A link that changes height (stairs, a ramp) or passes a door is a connector. Following an owner through a building teaches the route without anyone marking it.
- **Plan in legs.** For a target on another floor or in another building, plan over proven links and send ordinary moves leg by leg, checking arrival on the intended floor after each. A link that fails is marked and avoided until walked again.
- **Areas.** Group containers and stations by building and floor from the same graph. The storage plan needs these groups anyway.
- **Safety stays native.** Threat, water, fall and ward checks remain in the plugin for every leg.

### Not in scope

Changing the bounded walker's search, or centring on stairs and doorways: TKT-01M6M0HDCE27H971Q6SN08FHQN (Prefer clear building interiors and centered stair approaches) covers the native side.

### To decide first

- Where links are stored (memory.json beside landmarks, or the atlas) and how they age.
- Whether an ordinary multi-leg move is acceptable for unattended runs, or only when an owner started the run.

## Acceptance criteria

- [x] Completed walks become per-world links with height, persisted across restarts
- [ ] A target on another floor or in another building is reached by planned legs over proven links
- [ ] A failed link is avoided until it is walked again
- [ ] Containers and stations are grouped by building and floor
- [ ] Live: home to warehouse and back, and a census across both buildings, without hand-fed waypoints

## Implementation plan

Three slices, each its own pull request, deployed and merged as it completes.

1. **Record and inspect.** `internal/home/routes.go` keeps, per world, walked cells (1 m in x and z, 0.5 m in height so floors stay apart) and directed links between consecutive cells, with a count, last-walked time and failures. A link is reversible only when its height step is small, so a drop is never planned as a climb. `internal/memory/routes.go` samples session state while the body is grounded, in the world and not swimming; a teleport, portal or gap breaks the chain. Read-only debug from the first slice: `GET /api/routes` and `just vm routes` report cells, links, floors, connected groups and the newest recording. Deployed first, because the graph only grows from walking.
2. **Plan and explain.** A pure planner (`internal/route`) searches the graph from the cell nearest the body to the cell nearest the target and reduces the path to legs. `GET /api/routes/plan` and `just vm route-plan` print the legs, or why there is none: no walked cell near the start or target, or the two lie in separate groups, with the nearest reachable cell and the gap.
3. **Walk and learn from failure.** A runner sends ordinary `move_to` legs, checks arrival on the intended floor, marks a failed link and replans. The census and the home survey use it; each run's plan and per-leg outcome go to the audit log and behaviour status.

Unattended runs may use it (owner decision, 2026-11-17). Native threat, water, fall and ward checks stay in the plugin for every leg.

## Notes

**agent:claude-code/chests** at 2026-11-17T19:31:00Z

Owner decisions, 2026-11-17: Tas may use ordinary multi-leg moves on runs he starts himself, and may prefer them to the bursty bounded movement. Route memory is the top priority and blocks other work. Its debug information must be part of the regular process, so an agent can read it and guide the bot. Deploy and merge as the work completes.

**agent:claude-code/chests** at 2026-11-17T19:48:01Z

Slice 1 live on 2026-11-17: after four ordinary legs from the warehouse to home, just vm routes showed 30 walked cells and 58 links (29 walked, 29 inferred back) across height levels 31 to 32.5 m in one group, including the stairs. Criterion 1 ticked: persistence across restarts is covered by internal/home/routes_test.go; a live restart was not checked.

**agent:claude-code/chests** at 2026-11-17T20:29:05Z

Slice 2 (planner) is up as PR #184 and deployed to the VM. internal/route plans over the walked graph; GET /api/routes/plan and `just vm route-plan TARGET [FROM]` show the legs or the reason there is none. Live: home to the warehouse is 9 legs, 40 m, down the stairs; the way back climbs them.

Not done: nothing walks a plan yet. Slice 3 is the runner (ordinary move_to per leg, arrival checked on the intended floor, MarkRouteFailed and replan on a failed leg), then its use in the census and the home survey, with the plan and per-leg outcome in behaviour status.

**agent:claude-code/chests** at 2026-11-17T21:55:10Z

Slice 3, first part, is on branch t3/route-walk: the census walks plans over route memory (internal/behaviour/route_walk.go), one ordinary move per leg, arrival checked on the intended floor, a ground failure marking its link and the way planned again. Each entry and the run summary say how the walks went.

Live on 2026-11-17: a census started on home's upper floor crossed to the warehouse by itself ("route 7 legs 37 m; arrived"), and a later run opened 45 containers across both buildings with no hand-fed waypoints. Route memory grew from 42 to 134 walked cells.

Found on the way: a training dummy near home counted as a threat (it has an enemy faction and a MonsterAI). Built pieces are now never hostile, in perception and in native route checks. A timed-out inspection no longer ends the census.

Not done: the home survey does not use the walker; it moves with dry receipted moves in internal/goal/survey.go and needs its own change. The home to warehouse and back check as a plain owner move was not run; the census crossing covers the same ground one way. 20 containers stayed unchecked in the last run: four inspections timed out at 45 s each while a Greydwarf Elite stood about 37 m off, and the 600 s budget ended.
