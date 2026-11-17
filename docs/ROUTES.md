# Route memory

Tas's cautious bounded walker gives up inside buildings: on stairs, round a
double row of chests, between two buildings. An ordinary `move_to` gets
through the same ground. Route memory keeps where the body has actually
walked, so the controller can plan through a base over ground known to carry
it, and so an agent can see what is known when a move fails.

It is built in three slices, tracked on TKT-01M7NEG203A7HHNMQE0F8PW3AM (Route
memory: plan building routes over ground Tas has walked). **Recording and
inspection exist. Planning and walking do not yet.**

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

## Limits

- Recorded ground was walkable once. A door may since be shut, a wall built
  or a floor removed; the native walk still checks every step.
- A cell stands for one position. Two rooms divided by a thin wall inside one
  square metre share a cell.
- Nothing is recorded for ground the body has only seen.
