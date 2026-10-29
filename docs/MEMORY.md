# Remembered containers

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

This is the first slice of per-world memory. Recording notable resources,
missing-object inference, death causes/danger zones and drawing remembered
objects on the map remain on the broader memory ticket. Container snapshots
already come from the plugin's normal access/transfer path; storing them
requires only a controller deployment.

Use `just check controller test-packages ./internal/home ./internal/memory ./internal/behaviour ./internal/session ./internal/app ./internal/panel`
for the focused persistence, frame-scoping, freshness and API regressions.
