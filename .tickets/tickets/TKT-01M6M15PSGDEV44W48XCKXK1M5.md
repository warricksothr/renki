---
schema: 4
id: TKT-01M6M15PSGDEV44W48XCKXK1M5
title: Group household food targets and split panel configuration views
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/explore-night
  branch: t3/renki-portal-travel-startup
  worktree: /home/sothr/.t3/worktrees/renki/t3-6c0da0c0
  commit: 08590dec7bfa8ee607ca79d168fcfed85e39d9ad
  session: null
  claimed_at: 2026-11-04T19:41:14Z
  expires_at: null
archive: null
created_at: 2026-11-04T19:39:30Z
updated_at: 2026-11-04T20:05:26Z
created_by:
  id: agent:codex/explore-night
  name: ""
updated_by:
  id: agent:codex/explore-night
  name: ""
extensions: {}
---

## Description

Owner requests selecting several food types in one Household food plan, rather than duplicating goals, and splitting the growing panel into main operations, a shareable read-only map/video view, configuration sections and dedicated settings. Existing FoodStock already supports up to8 targets/minimum maps; current UI sends one entry. Expose editable target rows with independent refill/minimum bounds, preserve unsupported recipe/source honesty, show saved plans grouped rather than flattening each food into a goal. Keep map/video live while switching operator views and preserve active edits. Viewer access must be enforced by server with separate credentials and tightly allowed data/media routes; hiding buttons alone is insufficient. Existing private panel address is the default unless owner chooses public reachability; do not publish externally by inference. Review current adapter coupling before choosing a reusable monitor module.

## Acceptance criteria

- [x] One household plan accepts multiple distinct supported foods with per-food targets/minimums and displays them together.
- [x] Operator navigation separates operations, goals, places, people, audit/decisions and settings while preserving current controls and edits.
- [x] A shareable map/video view has separate viewer credentials and server-enforced access; viewer sessions cannot issue game or configuration commands or read audit/private settings.
- [ ] Meaningful backend and UI checks, full CI and committed review pass; deploy and verify current private output and Tas remain online.

## Implementation plan

Expose up to8 distinct household food target rows with independent target/minimum bounds, preserving the existing multi-target backend and source/recipe limits. Split operator sections into navigable views that stay mounted to preserve edits; keep a focused operations view and dedicated goals, places, people, audit/decisions and settings pages. Reuse the established map/video renderer in a monitor mode with no control components, rather than duplicate the renderer. Add a separate persisted revocable viewer credential; exchange fragment share credentials into a viewer cookie without exposing the operator token. Gate allowed map/status/media endpoints server-side, filter status and SSE to public observation fields, deny game/config/audit APIs, and allow only video playback negotiation mutations. Test raw HTTP denials, filtered streams, revocation/persistence and multi-target payloads plus operator/viewer UI navigation/media reconnection. Run fullCI, commit/review and deploy after Tas is safely home; retain a private-address deployment unless owner requests public publication.

## Notes

**agent:codex/explore-night** at 2026-11-04T19:54:59Z

Implemented one household plan with up to8 distinct supported food rows and independent target/minimum maps; existing goal cards already render the whole target mix together. Operator Operations,Goals & kit,Places & base,People,Decisions,Settings views stay mounted, preserving edits and one map/video instance. Viewer reuses map/video in monitor mode without hidden control components; Settings manages a separate random persisted revocable credential. Fragment exchange scrubs the link and sets a viewer-only HttpOnly/SameSite cookie. Server allowlist denies game/config/inventory/audit APIs, streams status only and strips chest contents from live/saved maps. Preview requests also use filtered data in operator browsers. Existing media can outlive revocation until disconnect; reconnect is denied, documented inUI/docs.

Focused desktop/mobile multi-food and viewer checks pass, including duplicate rejection,independent quantities,fragment scrubbing and no admin fetches. Backend tests cover persistence,failed-write rollback,rotation,cookie separation,raw HTTP denials,filtered preview and stream closure. Initial full panel run exposed old navigation assumptions in peel/status and shared settings-focus regressions; corrected scenario navigation passed focused rechecks. FullCI is running. Added just check vm viewer-check for live permission/revocation proof without logging credentials or changing game work. Sharing remains on the existing private address; no external publication.

**agent:codex/explore-night** at 2026-11-04T20:05:25Z

Full just check ci passed203s (.checks/20261104T200146Z-Ge5pgb.log), including66/66 desktop/mobile panel scenarios,Go race checks,actual plugin build and1033 native tests. The extra view-switch scenario retains an active video element/session through more than9s in Settings. Corrected new container test fixture dimensions and applied the plugin formatter; no production guard was weakened. Ready for committed review and deployment.
