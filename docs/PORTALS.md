# Building collision and portal knowledge

Local walking now caches actual collision faces from readable, non-convex
building meshes and tilted box colliders. Upright boxes and capsules retain
their cheaper primitive representation. A spatial hash selects nearby pieces;
their actual faces decide whether they obstruct the body's shared height band.
Door openings and arches stay open, ramps remain support, and walls on a
different floor do not fill the whole building's envelope.

Discovery prioritizes nearby geometry and the direction of travel. Unchanged
geometry is reused; transform, dimensions, mesh identity, trigger state and
enabled/lifetime changes invalidate it. Doors retain their existing live
state/interaction handling. Storage is limited to 2,048 triangles per collider
and 32,768 building triangles overall. Unreadable, convex, oversized or sheared
meshes fall back to live physics. Cached blockers can reject a route early;
only fresh support and body queries can admit movement. The cache is local to
the loaded scene and is rebuilt after reconnect, rather than trusting saved
collision faces from an earlier game run.

Each observed portal uses its network entity ID and carries its tag, connection
state, destination entity ID, optional known destination coordinates, and the
portal's allow-all-items setting. Destination coordinates come from an already
delivered network record. Renki does not request remote areas or infer a link
from matching tags. A connected portal may have an unknown destination.

Link observations share the existing per-world landmark memory and last-seen
timestamp. Newer retags, changed links and unlinks replace older observations.
They remain visible while the client is offline, with the same 30-day landmark
retention. The saved-atlas viewport loads remembered landmarks from the same
store, including after a controller restart; remembered chest markers also
retain their known contents. Map tooltips expose the observed destination
and item restriction.
Destination coordinates identify the remote portal, not a verified landing
point, safe terrain or an available return route.

The game teleports a player on entering a portal trigger; pressing interact
edits its tag. Ordinary walking avoids these triggers through cached bounds
and live capsule casts/overlaps, including before cache discovery. Other
trigger volumes are not treated as solid building parts. A final native input
guard also checks ordinary navmesh following and direct body-task movement,
which may bypass the dry route planner. A final guard refusal ends the owning
body task with `unreachable`, cancels its inputs and prevents later stuck
jump/strafe recovery from pushing into the same trigger. This does not authorize
portal entry or implement a deliberate return transition.

## Controlled travel next

Portal travel should be a distinct transition in a goal's route: walk to an
approach point, revalidate the exact source/destination link and native item
and server restrictions, deliberately enter, observe teleport completion and
the real arrival location, then resume walking with fresh local geometry.
Unknown destination, changed link, prohibited cargo, unavailable exit, failed
arrival and cancellation need separate outcomes. A remembered portal alone
does not authorize that transition. Automatic portal traversal and remote
arrival/return safety are not implemented by the observation work.

Use `just check plugin test-core BuildingMeshTests` for synthetic openings,
rotated faces, ramps and floor separation. Shared portal wire fixtures run in
both protocol suites; `just check controller test-packages ./internal/home
./internal/memory ./internal/protocol` covers durable links, unlink/retag,
unknown destinations, copy isolation and failed-save rollback. Actual base
navigation, portal discovery and trigger avoidance require native observations.
