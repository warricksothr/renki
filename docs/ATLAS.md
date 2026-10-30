# Durable world atlas: proposed storage and exchange design

Status: design/backlog, tracked by TKT-01M66Q5SFWV99JCGJXVARVRJH7 — Durable
versioned world atlas and portable mapping data. Nothing here changes runtime
retention, starts exploration, imports data or publishes a map. The first
implementation should persist the existing terrain backdrop; broader mapping
exchange and exploration build on that foundation.

## Current behavior and gaps

The plugin samples only loaded heightmaps: 16 m tiles, 25 corner measurements
at 4 m spacing, row-major toward +z. It reports height rounded to 0.1 m, biome
and a reference water plane. Unknown measurements are null. It prioritizes
nearby/ahead tiles, samples at most one tile per Unity update, refreshes known
samples after 30 seconds and missing samples after five seconds, and invalidates
nearby tiles after heightmap regeneration. This measures loaded terrain, not
personal visual coverage or route safety.

Both plugin and controller retain at most 256 tiles. The session resets its
terrain on reconnect/world/menu transitions; the panel reloads this session
snapshot. Thus the terrain backdrop does not accumulate durably across runs.
Landmarks, container contents, danger, player encounters and waypoints already
have separate persistent memory and retention policies in [MEMORY.md](MEMORY.md).

Two capture details matter before adding persistent expiry:

- Identical resampled tiles are suppressed. Receiving no change is not evidence
  that a particular tile was freshly checked, nor that it was forgotten.
- Unloaded samples carry their previous values forward. A resend or changed
  corner must not refresh the timestamps of those unobserved old values.

A durable atlas therefore needs measurement freshness/confirmation metadata,
not just timestamps attached to incoming complete-looking tile payloads.

## How Valheim changes

Most existing mountains, broad coastline shape and biome geography remain
stable during ordinary play. Relevant changes are usually local and layered:

- Digging, raising/levelling ground and landscaping change terrain height and/or
  surface appearance. Cultivation and paths can change surface paint without a
  meaningful height change. Today's height/biome stream does not capture paint.
- Buildings, bridges, walls, doors and gates alter access independently of the
  underlying ground. Pieces can be added, damaged or removed; door state can
  change in seconds. A bridge also introduces a surface above the heightmap.
- Trees can be felled or planted, ore/rocks depleted, crops grown and renewable
  pickables harvested and replenished. Resource presence and readiness have
  different lifetimes from the underlying biome/terrain.
- Players, creatures, drops, boats, weather, visibility and wave height are
  transient. The reference water plane currently reported is not a measurement
  of every wave or a guarantee about swimming conditions.
- Game generation changes, mods, world edits, resets and restored/diverged saves
  can cause wider changes or invalidate assumptions. Seed alone does not identify
  the current state of a shared world.

A fresh elevation tile can update the backdrop without proving that a doorway,
resource, roof or return route still exists. These need separate layers and
native revalidation at the time of action.

## Representation and identity

Keep observations in the game's local Cartesian coordinate frame:

- Horizontal coordinates are `(x, z)`, metres, +x east and +z north, with origin
  at game world zero. Height is the game's +y, positive up; game zero is the
  vertical datum, not a claim about a geodetic sea level.
- Use signed integer grid indices and floor division, including negative
  positions. Record tile size, spacing, quantization, units, axis order and
  schema version in the dataset manifest. Feature geometry uses `(x,z,y)` when
  encoded as conventional two-horizontal-plus-height ordinates.
- Identify a world with a configured server/world namespace, the game's world
  UID and an atlas epoch. The world name is display metadata, not the key. The
  available `ZNet.GetWorldUID()` is a 64-bit identifier and must cross JSON as a
  decimal string to avoid JavaScript integer precision loss.
- Keep game version, generator version and relevant mod/config fingerprint as
  compatibility/provenance metadata. Do not throw away all mapping merely
  because a game patch changes its version string. An incompatible generation
  change or a reset/fork/restore may need a new epoch or explicit reconciliation.
  Identical UIDs on diverged copies of a save do not justify automatic merging.
- Assign local source IDs and stream epochs/sequences. Imported, minimap/shared,
  loaded-heightmap, personally observed and traversed evidence stay distinguishable.

Keep the current 16 m/4 m bridge layout initially; changing transport layout is
not required to get persistence. Shared tile borders address the same global
sample key `(world, epoch, layer, resolution, gridX, gridZ)`. Store/merge that
sample once, then derive the overlapping 5x5 API tiles. This prevents two tile
copies from disagreeing about the same edge. Larger compressed storage pages
and coarse overview levels can be derived later without changing coordinates.

A 4 m heightmap cannot capture narrow trenches, walls, small step hazards,
bridges or multiple walkable floors. It remains a useful visual/survey layer;
collision geometry, navigation and directed route evidence remain separate.

## Storage decision

Use a separate local **SQLite** atlas, proposed as `atlas.sqlite`, rather than
adding an ever-growing raster to the whole-file `memory.json` rewrite path.
Keep the existing small memory stores during the first slice and join selected
layers for display/export; do not migrate every subsystem at once.

The pattern is a **current-state view plus a bounded change journal**:

- World/source manifests describe identity, coordinates, compatibility and schema.
- Canonical samples/pages contain terrain values, nodata masks and per-sample
  observation references. Index fixed-grid addresses by world/layer/resolution
  and integer grid/page coordinates.
- A current-state table points to the accepted value/observation for each sample
  or object. Keep its actual measurement time, trusted confirmation time,
  received time and resolution/provenance separately.
- Content-addressed payloads and changed revisions retain old state for a bounded
  period. Hash canonical values/masks/encoding, excluding mutable confirmation
  timestamps, so repeated confirmation does not create identical revisions.
- Feature layers use explicit object IDs/geometries and observation/tombstone
  records where removal is actually established. Spatial R*Tree indexes become
  useful for feature bounding boxes; the regular terrain grid can start with
  ordinary indexed integer coordinate ranges.
- Derived viewport tiles, contours, shoreline rendering and coarse overviews are
  caches. They can be rebuilt from retained accepted evidence.

Use transactions, one bounded batch writer and versioned migrations. WAL mode
allows panel reads during writes on a local filesystem; it is not a shared
network-drive synchronization mechanism. Export a consistent snapshot through
SQLite's supported backup/snapshot mechanisms rather than copying only a live
main file while its WAL contains uncheckpointed changes. The portable export
format is separate from the writable database.

Deleting expired rows frees reusable database pages but need not shrink the
file. Include bounded checkpoint/compaction work and both logical-record and
physical-disk limits. An always-growing event log or one complete atlas copy
per scan would defeat the retention requirement.

At current transport density, one square kilometre has about 3,906 16 m tiles.
Twenty-five float32 heights plus one-byte biome codes per tile are about 0.49 MB
of raw values per square kilometre, before masks, metadata, indexes and history.
This counts duplicated transport borders; canonical storage avoids those
copies. It is an order-of-magnitude estimate, not a storage guarantee. Benchmark
real schema/page overhead, compression and change density before choosing quotas.
Store only acquired data; do not generate a whole world from its seed.

## Merge and refresh semantics

Accumulate observations spatially; do not overlay whole screenshots or replace
the entire atlas with the latest small sensor snapshot.

1. Validate world/epoch, coordinate schema, source, stream ordering, bounds,
   payload size, resolution and measurement metadata before ingestion.
2. Capture actual fresh measurements separately from carried cache values. A
   per-sample fresh/valid mask and bounded measurement age from the producer's
   monotonic clock can let the receiver establish confirmation age without
   blindly trusting a VM wall clock. Null/unloaded or merely resent values do
   not renew confirmation. Emit compact positive confirmations for unchanged
   fresh samples on the existing refresh schedule, without extra world sampling.
3. Accept a newer compatible trusted local observation at the same sample key.
   Preserve prior values where the new observation is unknown. For changed
   values, atomically journal the old accepted revision and update current state;
   for equal values, renew genuine confirmation metadata without a new revision.
4. Keep freshness per sample/layer. A changed corner or a new structure cannot
   make all adjacent old terrain or other object contents appear newly observed.
   Unknown is distinct from a positive observation that an object was removed.
   Incomplete scans alone cannot create object deletion tombstones.
5. Invalidate derived caches and route candidates whose dependencies intersect
   accepted changes. Do not invalidate the whole world for one ground edit, or
   call a route safe because no height sample changed.
6. Compact/reclaim expired changed revisions and current samples according to
   their layer's policy. Expired/evicted current state does not fall back to an
   old historical revision and silently resurrect it.

Ordering from one producer can use stream epoch/sequence plus sample freshness.
Sequence numbers from different producers or sessions are not globally ordered.
For later imports, record both observation and ingestion time and preserve
conflicts/provenance. Clock skew, differing resolution, incomplete observations,
changed world epochs or unrelated sources make blind global last-write-wins
unsafe. A local receiver's sequence order is not proof that an imported map was
measured more recently. Initial import can stay read-only until compatibility
and merge policy are explicit; no distributed CRDT framework is needed initially.

## Retention policy to validate

Use per-layer settings, with a hard overall byte/record budget. Suggested initial
values for the atlas implementation, **not runtime defaults installed now**:

- Current terrain/biome samples: mark stale after 30 days without actual fresh
  confirmation, hard-expire after 180 days. Preserve unchanged confirmed samples
  without storing duplicate payload versions. Recently used data may be preferred
  for capacity eviction, but reading a sample never renews its measurement age.
- Replaced terrain revisions: at most 30 days and eight revisions per sample/page,
  whichever cap is reached first. History expiration is independent of current
  sample confirmation; it must not erase a recently confirmed latest value.
- Static structure/resource locations: retain historical knowledge on an order of
  weeks, but recheck existence/access/readiness before use. Doors, chest contents,
  resource availability and supporting floors need substantially fresher checks.
- Live players/creatures/conditions: seconds for a current overlay; dedicated player
  history and danger continue their existing separate fading policies. Do not
  bake them into long-lived terrain pixels.
- Imported maps: use their original measurement age and provenance, not import
  time as freshness. Unknown measurement age is explicitly unknown and cannot
  earn a fresh local observation or extend a navigation validity period.

A configurable TTL bounds age; a storage quota bounds frequently edited or
heavily explored worlds. Lower-resolution overview retention, if later added,
needs its own explicit lifetime and known resolution instead of turning expired
fine detail into fresh-looking coarse detail. Exports need an as-of time and
expiry metadata; local expiry cannot revoke copies already distributed elsewhere.

## Portable formats and broader sharing

Use established tools at the appropriate boundary:

- **SQLite + spatial indexes** for incremental local writes, bounded history,
  transactional merges and viewport queries. A generic SQLite database is not
  automatically an OGC GeoPackage; that is a separate compliant export schema.
- **OGC GeoPackage** for a portable GIS snapshot of selected features and map
  layers with explicit local coordinates. It supports spatial-reference metadata
  and defines undefined Cartesian SRS -1. A named engineering frame plus manifest
  is preferable where tool support allows. Numeric elevation needs the appropriate
  gridded-coverage representation/extension rather than pretending a colored
  raster image preserves height measurements. GDAL/QGIS are useful validators.
- **Cloud Optimized GeoTIFF (COG)** for exported numeric terrain/biome rasters with
  nodata masks and overview levels, when range-readable larger raster snapshots
  are useful. Declare the local coordinate frame; keep observation metadata in
  companion bands/catalogue records. This is not the live transactional store.
- **PMTiles**, later, for immutable tiled publication over HTTP/object storage.
  It is read-only, so produce a new versioned archive rather than rewriting
  live observations into it. Tile-matrix/projection assumptions must match the
  reader; existing web-map clients often assume Earth/Web Mercator conventions.
- **Leaflet CRS.Simple** or an OpenLayers custom projection for a future common
  map viewer in game coordinates. The current canvas can remain while storage
  changes; a viewer migration is not a prerequisite for persistence.

RFC 7946 GeoJSON uses WGS84 longitude/latitude in degrees. Raw Valheim `(x,z)`
metres must not be labelled as compliant GeoJSON or EPSG:4326/3857 just to make
a tool accept them. Use a local-coordinate manifest/format or a deliberate,
documented export transformation; never change navigation coordinates to match
a globe viewer.

Start sharing with an explicit selected-layer export and a versioned manifest:
world namespace/epoch, coordinates/resolution, schema/encoding versions, source
provenance, observation intervals, hashes, and bounding boxes. Terrain and biomes
can be a simple first dataset. Player histories, private waypoints, chest contents
and server connection details are independent optional data, not implicit parts
of a terrain export. Exporting data does not grant command/control authority.
No upload, public service or automatic federation is authorized by this design.

If multiple observers eventually contribute, exchange idempotent bounded batches
and manifests, validate compatibility, retain source history/conflicts and derive
a current view. A public read service or static snapshot archive can be added
without exposing or sharing the live SQLite file. PostGIS is an option when
actual concurrent server ingestion/spatial queries warrant it; it is unnecessary
for one bot's local atlas.

## Incremental delivery and proof

1. Extend capture with stable world/epoch metadata and honest per-sample freshness
   or refresh confirmations. Older producers without those guarantees stay
   session-only; do not silently persist by display name or reset their age.
2. Add the local SQLite atlas and bounded batch ingestion, merge/history/retention,
   migrations, quotas and recovery. Keep the current session cache as the hot
   working set; the atlas is the durable source behind it.
3. Add bounded viewport/overview queries, remembered/stale styling, per-point
   age/source hover and explicit saved-world selection while Tas is offline.
   A selected saved atlas is not a live connection; route tasks continue to use
   their existing live-world and native checks.
4. Verify restart restores the same accumulated map, a real localized terrain
   edit replaces only the appropriate observations, unchanged confirmations do
   not add history, and unloaded/expired/cross-world data does not masquerade
   as new evidence. Then add portable selected-layer snapshots/import previews.

Replay/storage tests need partial overlapping tiles and border identity,
negative coordinates, holes/nodata, reordering/duplicates, stale cached resends,
equal-but-fresh measurements, clock skew, conflicting imports, same-name different
worlds, diverged epochs, save/crash/migration failure, TTL/quota boundaries,
history non-resurrection and physical maintenance. UI checks need offline saved
maps, viewport limits and world/connection changes during requests. Live terrain
change validation can use ordinary owner landscaping when it naturally occurs;
no autonomous ground edit should be performed merely to test persistence.

Persistent mapping is useful before exploration goals and does not itself start
them. [EXPLORATION.md](EXPLORATION.md) still defers expedition behavior until M2
and M3, and distinguishes imported/map knowledge from personal survey success.

The implementation backlog remains draft, in dependency order:

- TKT-01M66QF9EH8W2EG78AKKKF05PP — Report stable world identity and truthful terrain freshness.
- TKT-01M66QF9G576JFCQ0JBP3RWR2E — Persist a bounded world atlas with merge and expiry; depends on capture.
- TKT-01M66QF9HTVA5EEF2ASPEQYA65 — Show saved and stale atlas layers in the map; depends on storage.
- TKT-01M66QF9KD620257SS5A0W987P — Define portable atlas exports and validated import previews; depends on storage.

## References

Reviewed on 2026-10-30:

- [SQLite WAL: concurrency and local-filesystem constraints](https://sqlite.org/wal.html).
- [SQLite R*Tree: spatial bounding-box indexes](https://sqlite.org/rtree.html).
- [OGC GeoPackage 1.4: container, CRS, features, tiles and coverage extensions](https://www.geopackage.org/spec140/index.html).
- [RFC 7946: GeoJSON coordinate reference system](https://www.rfc-editor.org/rfc/rfc7946#section-4).
- [GDAL COG: tiled raster and overview export](https://gdal.org/en/stable/drivers/raster/cog.html).
- [PMTiles concepts: immutable tiled archives and HTTP range access](https://docs.protomaps.com/pmtiles/).
- [Leaflet maps in non-geographical coordinates](https://leafletjs.com/examples/crs-simple/crs-simple.html).

Current capture/validation behavior is also established by
`plugin/Renki.Core/Perception/TerrainCache.cs`,
`plugin/Renki.Plugin/TerrainReporter.cs`, `internal/session/terrain.go`,
`internal/protocol/terrain.go` and their existing tests. The installed game API
confirms that world UID and generator-version metadata exist; no decompiled
game source is part of this proposal.
