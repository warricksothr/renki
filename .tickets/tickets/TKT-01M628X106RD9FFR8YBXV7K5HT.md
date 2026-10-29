---
schema: 4
id: TKT-01M628X106RD9FFR8YBXV7K5HT
title: Web control panel to watch and steer the bot
type: epic
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - tooling
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-resume
  branch: t3/terrain-map
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: d424ebfc435272fb8b4331b9c6f6a204e734a30f
  session: null
  claimed_at: 2026-10-29T21:29:05Z
  expires_at: null
archive: null
created_at: 2026-10-28T22:08:14Z
updated_at: 2026-10-29T21:38:43Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

A web page, served by the controller, for watching the bot live and steering
it: where it is, what it is doing and why, what it heard and said, and
switches for its behaviours. It is reached over Tailscale at the VM's
tailnet address and asks for a login token, since anything on the tailnet
can reach it. The audit log (TKT-01M627TCX) is its source for history.

## Acceptance criteria

- [ ] The panel opens from a tailnet device, behind a token
- [ ] It shows the bot's position, task, chat and recent decisions live
- [ ] It can stop the bot, send it home and log it out
- [ ] Behaviour switches change what the bot does without a restart

## Implementation plan

Implement the terrain/location map direction approved by the owner on 2026-10-29. This extends the existing panel epic; unrelated draft exploration/navigation tickets stay draft.

Add additive terrain protocol messages with 16 m tiles, a 4 m corner grid, nullable loaded-heightmap samples, biome and water level. A bounded Core cache schedules nearby/ahead tiles, samples at most one 25-point tile per update, refreshes periodically, sends only changed tiles, resends after reconnect and invalidates affected tiles after Heightmap regeneration. Missing heightmaps remain unknown. Cache bounds and work budgets are regression-tested. Terrain is a visual observation layer, not navigation proof; persistent exploration remains separate.

Add exact current-location telemetry (ground height below the character, supporting surface, in-water/swimming and water depth at feet), avoiding dry-land inference from grounded. Keep cached terrain scoped to the current connection/world, expose authenticated map snapshots, and bootstrap the browser without replaying old-world tiles. Use persisted landmarks as faint last-seen markers, overridden by live perception.

Build the approved visual hierarchy: biome/water terrain shading, shoreline and height contours, location/height/support readout, clear facing marker, perception-range ring, compass and scale. Provide terrain/elevation/memory/resources switches; retain normal navigation and task markers. Move routine scan/cache traces behind a Diagnostics filter. Verify desktop/mobile views, unknown terrain, negative tile coordinates, null samples, world/reconnect reset, cache reuse/invalidation, authentication and terrain edits where live staging permits.

Run focused just checks, just check ci, committed just check review, then land and deploy the plugin/controller. Keep outstanding staged movement/combat checks deferred; don't turn visual terrain into a route-safety claim. Record live sampling costs and retained limitations here.

## Notes

**agent:codex/renki-resume** at 2026-10-29T21:38:43Z

The owner approved the terrain/location map direction. Implemented on t3/terrain-map: loaded-heightmap biome/elevation tiles (16 m tiles, 4 m corner samples), bounded 256-tile plugin/controller session caches, at most one 25-point tile sampled per plugin update, near/ahead priority, known/missing refresh, changed-only sends, welcome resend and Heightmap regeneration invalidation. Unknown samples remain null and unloading preserves earlier measurements. State precedes the first terrain patch after welcome. Shared Go/C# fixtures document the additive wire fields.

The location readout now distinguishes character elevation, terrain height, support (terrain/structure/ship/etc.), water above feet, swimming and shelter. GET /api/map is authenticated and scoped to the live connection/world; it supplies cached tiles, current perception/walls and at most 512 nearest persisted landmarks. The browser refreshes every 5 s, clears on connection/world/menu changes, rejects replies from old view generations and draws last-seen landmarks faintly with original timestamps, suppressing live duplicates.

UI adds biome/water backgrounds, interpolated shorelines, 5 m contours, terrain hover details, layer switches, facing marker, perception range, compass and scale. Resources start hidden and the long marker key is collapsed. Routine scan/cache traces move to an unchecked Diagnostics filter. Desktop and 390 px browser checks passed with no console errors or horizontal overflow; layer switches, unknown lookup and world-reset clearing passed. The fixture preview uses the actual assets, has no game/network actions, and can load recorded API snapshots through just controller preview-map. Node.js 20+ is now required for just controller test-map, included in just ci.

Cache/policy tests, interpolation/clipping tests, scope/bounds/API tests and the shared protocol regressions passed. Full gate passed with race checks, both builds, plugin build and 165 C# tests after repairing the generated epic index through just tickets-fix. Final gate/review and live deployment follow. The broader panel epic remains in progress: behaviour-instance switches are separate work.

Remaining staged game checks: compare a shoreline/contours to the actual terrain, landscape a safe area and verify regeneration refresh, and check wading/swimming/return-to-land labels with the owner present. Terrain is coarse and cannot establish path safety, floors/bridges/caves or actual wave surfaces. Terrain is cached for the session rather than persistent explored-map memory. No harvesting or combat check is performed for this visual change.
