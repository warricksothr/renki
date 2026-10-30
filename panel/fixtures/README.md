These assets were extracted losslessly from PR #102's recorded preview
(`panel-recorded.html`, parent review 2026-10-30, commit 9517fed): status,
134-tile map, saved atlas and 960×540 JPEG. They are recorded game observations,
not current observations. Inventory, activity, settings and camera timing are
simulated by the existing `scripts/map-preview.py` transport. No credentials or
live endpoints are needed. Goal failures and remembered chest contents remain
as recorded; nothing in these examples proves a new game outcome.

The suite freezes time at 2026-10-30T19:00:00Z, seeds Math.random with 102 and
loads the locked Inter Latin WOFF2. Camera age/expiry are simulated. Production
assets and the shared map renderer are embedded on each run, never copied from
an old HTML screenshot. Fixture changes are ordinary reviewed source changes.
