# Route memory

Tas's cautious bounded walker gives up inside buildings: on stairs, round a
double row of chests, between two buildings. An ordinary `move_to` gets
through the same ground. Route memory keeps where the body has actually
walked, so the controller can plan through a base over ground known to carry
it, and so an agent can see what is known when a move fails.

It is built in three slices, tracked on TKT-01M7NEG203A7HHNMQE0F8PW3AM (Route
memory: plan building routes over ground Tas has walked). **Recording,
inspection and planning exist. Nothing walks a plan yet.**

## What is recorded

The controller samples the body's position four times a second. Whatever
moved it counts: an order, a behaviour, or following an owner through a
building. Following someone up the stairs is how a route is taught.

- The world is cut into **cells** one metre square and half a metre high, so
  two floors of a building never share a cell. A cell keeps the last exact
  position walked inside it. That is where a planned leg would be sent, not
  the cell's centre, which may be inside a wall.
- Two samples become a **link** when the body was on the ground at both, in
  one connection, at most three seconds and three metres apart, and moved at
  least half a metre. In the air, in water, dead, in a menu, after a teleport
  or portal: the chain breaks and nothing is joined across it.
- A link is recorded in the direction walked. When its height step is at
  most 0.75 m the way back is added as an **inferred** link. A bigger drop
  stays one way, so a fall is never planned as a climb.
- A link counts how often it was walked and how often a planned move along it
  failed. Nothing marks failures until the walking slice exists.

It is kept per world in `memory.json` beside landmarks, written in batches
every 30 seconds and at shutdown. A world keeps at most 20,000 cells, and
ground not walked for 60 days is forgotten with its links.

## Reading it

    just vm routes

prints the walked cells and links, the height levels walked, the separate
**groups**, the time of the newest recording, and whether the body stands on
walked ground. A group is a set of cells with a walked way between them.
More than one group means some place has no known route to another: an upper
floor Tas has stood on but never walked up to shows as its own group.

`GET /api/routes`, behind panel authentication, returns the same as JSON:
`world`, `stats` (`cells`, `links`, `walked`, `failed`, `levels`, `groups`,
`newest`) and `body` (`pos`, `cell`, `known`, `nearest`, `nearestM`).
`?cells=1` adds every cell with its links. It sends no game task.

Read this first when a move fails or a route cannot be planned. It shows
whether the target's surroundings were ever walked, and on which level.

## Planning

    just vm route-plan home
    just vm route-plan "new home storage west"
    just vm route-plan "-163.35,30.97,-467.12"
    just vm route-plan home "-169.69,31.03,-467.8"     # from somewhere else

plans from the body, or from a given position, to a position or a named
place: `home`, or a saved place by name. It is read-only. It prints the legs,
each an ordinary move, or why there is no route and what to walk to make one.

A position joins the graph at the nearest walked cell within 4 m and 1.25 m
in height, so it joins on its own floor: the room above is not "near". The
planner then takes the shortest walked way. A link never walked in that
direction costs a quarter more. A link a move failed along is left out until
it is walked again in that same direction; walking it the other way proves
nothing, since a gate may open from one side only. The first leg goes to the
walked cell the body joined at, when that is more than half a metre away.

The way is cut into **legs**. A leg ends where going on would take the walked
way more than 0.6 m off a straight line, with height counted double, or make
the leg longer than 10 m. So corners, stair feet and stair tops end legs, and
a straight corridor is one leg. When the last walked cell is short of the
target, a final leg goes to the target itself and is marked off walked ground.

When there is no route, the reason is one of:

| Reason | Meaning | What to do |
|---|---|---|
| `nothing_walked` | The world has no walked ground. | Walk the body, or have it follow someone. |
| `start_off_walked_ground` | No walked cell near the body on its level. | Move it onto known ground. The report names the nearest cell. |
| `target_off_walked_ground` | No walked cell near the target on its level. | Walk there once. |
| `no_walked_way_between` | Both ends are known but not joined. | Walk the gap. The report gives the joined cell nearest the target and the distance. |
| `unknown_place` | The place name matches no saved place, or no home is saved. | Use one of the names the report lists. |

`GET /api/routes/plan?to=x,y,z` or `?place=NAME`, with optional
`from=x,y,z`, returns `found`, `reason`, `explain`, `start`, `target`, `legs`
(`to`, `lengthM`, `climb`, `cells`, `inferred`), `lengthM`, `cells`,
`avoided`, and for a missing route `closest` and `gapM`.

## Walking a plan

A run that has to get somewhere the bounded walker cannot reach walks a plan:
one ordinary move per leg, checked on arrival (`internal/behaviour/route_walk.go`).

- A leg has arrived when its move succeeded and the body stands within a
  metre of the leg's stop distance and within a metre of its height. A move
  that reports success from the floor below has not arrived.
- A leg that fails for a reason about the ground (`unreachable`, `stuck`,
  `blocked`, `deep_water`, or a success that stopped short or on another
  floor) marks the link out of the walked cell nearest where the body stopped
  as failed, in that direction. A timeout, a refusal or a stop marks nothing. The way is then planned again
  from where the body stands, at most twice. `just vm routes` counts failed
  links; a plan says how many its search skipped.
- With no plan, the run makes the one direct move it would have made without
  route memory. That move is recorded like any other, so it teaches the way.
- A walk ends early once the body is within the stop distance of its target.

The container census uses it for every walk it makes: to a verified standing
position, or toward a container too far away to be loaded. Each container's
entry carries a `route` line and the run's summary a `Walks:` part, such as
`route 9 legs 40 m; arrived` or
`route 4 legs 18 m; leg 2 of 4 failed (unreachable), link marked failed; route 6 legs 25 m; arrived`
or `no route (target_off_walked_ground); direct move failed (timeout); did not arrive`.
Read it before guiding the body by hand: it says which leg failed and why.

### Going to a place

`Tas, go to <place>` in chat and the panel's go button take a trip over walked
ground when route memory has a plan of two legs or more: the `travel`
behaviour, saved as `Owner travel` (docs/BEHAVIOURS.md). Its status says how
the walk went, and Tas says why in the game when he does not arrive. With no
such plan, or while Tas is busy with something else, they send the one direct
move they always did. That move still teaches the way.

### Putting things away

Stow keeps its native way to a chest: a dry move, or the deposit's own
bounded walk. When that fails `unreachable`, `stuck` or `blocked` and route
memory has a plan of two legs or more to the chest, stow walks it with
ordinary moves, stops within 2.5 m and asks again. It does this once for each
chest; a walk that does not arrive is that chest's failure, with the walk's
story as the reason. Collect, gather, marker food and marker cargo all put
things away through stow, so they have it too. Marker cargo's return to its
marker falls back the same way.

The same fallback, tried once and only for those three reasons, covers the
other moves these runs make to a place (`detour` in
`internal/behaviour/route_walk.go`): collect's move to its site and back,
marker food's return to its marker after foraging or cooking, the day rest's
move to the bedside, and gather's walk home, its move to the work point and
its return there. Gather's move toward a tree is left alone: a tree stands on
ground not yet walked. When there is no known way the run ends as it
did before, with the native reason.

A fallback walk is made of ordinary moves: it does not avoid hostiles or
water as the native move does. The run's own threat and injury checks still
stop it between legs.

### For other runs

The walker takes its plans from a `Navigator` (`Plan`, and `Failed` for a leg
that did not carry the body). `RouteMemory` is the only one so far; a planner
over ground Tas has seen but not walked would be another. A `MoveStyle` sets
how each move is made: plain, or dry and avoiding hostiles, running or not,
and the time each leg is given.

A run that keeps its own state, or must survive a restart, uses `NextLeg`: it
plans from where the body stands and returns the first move, with nothing to
remember between moves. `Arrived` judges a move and `LegFailed` reports one
that failed.

Not converted yet: go home, the corpse run, retreat, the food goals and the
home survey.

## Limits

- Recorded ground was walkable once. A door may since be shut, a wall built
  or a floor removed; the native walk still checks every step.
- A cell stands for one position. Two rooms divided by a thin wall inside one
  square metre share a cell.
- Nothing is recorded for ground the body has only seen.
