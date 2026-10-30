# World memory

## Last-seen places

Useful structures, pickables (including berry bushes) and recognized ore
deposits persist per world in `memory.json`. Each entity keeps its location
rounded to 0.5 m, last-seen wall time and portal tag when present. Prefab,
kind and tool tier are shared in a per-world catalogue. Door open state,
characters, drops, tombstones and individual scenery trees/ordinary rocks
are not part of this durable landmark layer. Ore recognition currently
uses known vanilla prefab families; unknown/modded rocks are excluded.

The recorder samples new perception frames once per second and batches
writes for 30 seconds. Graceful controller shutdown waits for the final
batch; an abrupt process exit can lose that interval. Failed writes keep
the pending batch with original timestamps. World changes flush the old
batch before collecting the new world; if that save fails, new-world
collection waits for persistence to recover. Each batch holds at most
16384 distinct entities, and each world retains at most 16384 landmarks.
Updates prune entries older than 30 days, evict the oldest excess and
remove unused catalogue metadata. Missing entities in a snapshot do not
erase remembered places or refresh their last-seen time.

`GET /api/memory/landmarks`, behind panel authentication, returns the current
world and expanded `landmarks` with `entity`, `kind`, `prefab`, `pos`, `at`,
`ageS`, optional `minTool` and optional `tag`. A negative age means a
future-dated observation. This API describes **last seen** places: it is
not a guarantee of existence, harvest readiness, chest contents or access.
Consumers must revalidate before acting. It does not send game tasks.

Missing-object inference needs explicit coverage information first:
perception truncates each kind to its nearest results, and the resource
collider scan can saturate. Absence therefore does not prove destruction
or harvesting. The panel combines these remembered landmarks with the
current perception and the danger layer described below.

## Remembered containers

The controller retains exact chest observations in `memory.json`, beside
homes and graves. Each world has its own entity IDs. An observation keeps
position, size, items and the wall-clock time it arrived; the prefab is
retained when perception identifies it. A later observation replaces the
whole contents, including an empty chest.

A recorder batches new session observations once per second. It does not
open containers or send game tasks. Failed saves leave memory unchanged
and are retried with the original observation time. Reconnects and world
changes reset the recorder's live frame tracking. Each world retains at
most 4096 containers; new batches prune observations older than 30 days
and evict the oldest excess records. Observation history is not kept.

`GET /api/memory/containers`, behind panel authentication, returns the current
world and its remembered containers with `at` and `ageS`. A negative age
means the observation is future-dated relative to the current clock.
These are remembered contents; players may have changed a chest since.
Room calculations reuse `Inventory.Room`, with the same upper-bound
limitations for differing item qualities; they do not account for weight.

Behaviour stock conditions can use these observations after a controller
restart. Their explicit `maxAgeS` still applies, and future-dated snapshots
fail. Live snapshots override remembered contents, even when empty or
insufficient for the condition. Stow must open the chest and use current
contents before transferring items.

Container snapshots come from the plugin's normal access/transfer path.
Both memory recorders require only a controller deployment.

Use `just check controller test-packages ./internal/home ./internal/memory ./internal/behaviour ./internal/session ./internal/app ./internal/panel`
for the focused persistence, frame-scoping, freshness and API regressions.

## Injuries and danger

Actual `damaged` and `died` events persist per world with position, wall time,
amount and the attacker ID/prefab when supplied. New plugins report the exact
injury position; older ones use the last state position. Unidentified causes
remain unknown rather than being labelled as falling or drowning. An event
needs current-connection in-world state received within five seconds and game
time within three seconds after that state. Duplicate injury sequences are
ignored. Scope is captured before dispatch, so a reconnect cannot attach an
old event to another world. A zero-HP in-world state can still locate a death.

The bridge queues up to 256 injuries without doing disk IO; a once-per-second
worker writes batches, retains failed batches with their original times, and
flushes at graceful shutdown. A pending batch is also capped at 256; overflow
is counted and logged. An abrupt exit can lose queued injuries, and a failed
final save is logged. Each world retains at most 256 recent injuries. Writes
prune injuries older than seven real days; reads also omit expired/future
observations. Save failure rolls back the entire cross-world batch.

The map and resource selection derive up to 128 twelve-metre zones from these
records. Injuries within eight horizontal metres and four vertical metres of
a zone's newest injury are grouped. Ten damage adds one score unit, a death
adds five, each contribution halves every real day, and zone scores cap at ten.
The optional **Danger** map layer shows translucent red circles; hovering the
area shows injuries, deaths, height, most recent attacker and last-hurt time.
These are remembered hazards, not sightings of a currently living enemy.

Initial `gather` tree choice and `collect` pickable/drop choice add sixty metres
of equivalent travel per unit of local danger score, tapering to zero at the
zone edge. This favours safer available work and fades back toward nearest
selection as danger decays. Subsequent tree-splitting tasks still finish the
selected trunk chain. Explicit site travel, delivery, return-home and native
navigation do not yet route around danger; a safer target does not prove a
safe path. Combat remains governed by its existing permission gate.

## Player encounters

Player sighting/heard-location histories share `memory.json` but have their own
age, precision and retention policy, described in [PLAYERS.md](PLAYERS.md#remembering-and-locating-other-players).
These observations supplement the static landmark layer; they do not add
characters to that layer or give it a claim about current player positions.
The recorder samples new fresh perception once a second, records located
normal/whisper/shout chat before presence/command handling, and batches disk
writes and aging sweeps every 30 seconds. Bridge callbacks never do disk IO.
At most 4096 observations await persistence; overflow drops new history entries
while retaining the bounded recent-position cache. A current requester position
expires after 30 seconds and cannot come from disk or a previous connection.

Authenticated `GET /api/memory/players` exposes `world` and bounded
`encounters`, newest first, including unsaved observations. Each encounter has
`name`, `pos`, `at`, `source`, `resolutionM` (cell width) and `timeResolutionS`.
Reads omit future/expired evidence and coarsen older records immediately. This
is a read-only history endpoint and sends no game tasks. Like the existing
memory APIs, it reports the current in-world scope, or an empty result offline.
Fading applies to operational encounter memory; retained audit files remain
independent diagnostic records, with their original observation precision.
