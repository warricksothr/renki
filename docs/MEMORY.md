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
or harvesting. Death causes/danger zones and map rendering remain on the
broader memory ticket.

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
