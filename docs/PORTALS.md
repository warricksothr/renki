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

## Following a leader through a portal

An ordinary `follow` can copy a leader's portal departure. The plugin observes
players even while passive work owns the body. If a leader disappears within
2 m of one connected portal, or their position jumps more than 32 m from it,
the follow walks into that source's trigger. The observation expires after
8 s, so a follow returned promptly after a passive pickup can use it. Missing
or ambiguous portal evidence retains the ordinary lost-player behavior.

Only the observed source and its unchanged native destination are admitted.
The allowance lasts for the active crossing's physics step, including cached
obstacle checks, planning probes and the final input guard. Other portals
remain blocked. Cancellation, replacement, failure and teleport revoke it.
Walls, support, dry-ground and native cargo/server restrictions still apply.
The bot enters the trigger normally; it does not call teleport directly or
edit tags. The native target record must be ready before entering.

Approach is limited to 45 s, refusal inside the trigger to 4 s, teleport to
60 s and looking for the leader after arrival to 15 s. The ordinary five-second
lost-player timer does not run during the transition. Arrival clears local
geometry and restarts following with a fresh path. `portal_cargo` explains
that the cargo cannot cross; other crossing failures use `portal_failed`.
Both ask the owner to return and avoid the failed-follow trip home.

Use `just check plugin test-core PortalFollowTests` for departure expiry,
remote position jumps, exact-link admission, cargo refusal, phase limits,
cancellation and return-trip policy. Controller tests cover the prompt return
of a follow after its passive anchor disappears, and owner-facing failure
messages without a trip home. A supervised portal-pair crossing and return
with Nimah remains required live evidence; offline policy tests do not prove
native trigger entry or arrival.

## Explicitly ordered crossing

`interact` with `entity` and `portalTo` orders one native crossing, distinct from
text interaction or walking. The controller requires `portal_travel_v1`; both
IDs pin the source and destination. The source must be loaded within 16 m. It
rechecks the exact link, ward access and native cargo rules during approach,
walks on supported dry ground with hostile avoidance, and admits only this
trigger. Other interaction modes cannot be combined with it. Cancellation
revokes admission; native refusal and bounded entry/teleport/arrival timeouts
retain their portal failure outcomes. It never invokes teleport or edits tags.

Success requires observed native teleport completion followed by a grounded,
non-swimming position within 4 m in three dimensions of the loaded expected
exit, with at most 1.5 m vertical difference. Geometry resets on teleport. A task/audit receipt records source,
destination, actual arrival and the separately observed exit's return link;
an observed return link alone does not prove a safe or successful return.
Durable map route planning and repeatable native return evidence remain open.

## General controlled travel next

Portal travel should be a distinct transition in a goal's route: walk to an
approach point, revalidate the exact source/destination link and native item
and server restrictions, deliberately enter, observe teleport completion and
the real arrival location, then resume walking with fresh local geometry.
Unknown destination, changed link, prohibited cargo, unavailable exit, failed
arrival and cancellation need separate outcomes. A remembered portal alone
does not authorize that transition. General portal route planning and remote
arrival/return safety are not implemented by leader-following.

Use `just check plugin test-core BuildingMeshTests` for synthetic openings,
rotated faces, ramps and floor separation. Shared portal wire fixtures run in
both protocol suites; `just check controller test-packages ./internal/home
./internal/memory ./internal/protocol` covers durable links, unlink/retag,
unknown destinations, copy isolation and failed-save rollback. Actual base
navigation, portal discovery and trigger avoidance require native observations.
