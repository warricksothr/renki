---
schema: 4
id: TKT-01M67175263Z2YDVHSMRN6WWRF
title: Capture bounded game-camera JPEGs for visual debugging
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
  - protocol
  - tooling
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T18:30:10Z
updated_at: 2026-10-30T19:18:44Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

The owner requests an updated JPEG of the bot's rendered game view every five seconds, configurable with a ceiling of one capture per second, for the panel and automated visual debugging. Implement native Unity capture at a bounded resolution/quality and one capture in flight. Use a latest-only media slot, authenticated retrieval with scoped metadata, runtime controls and no embedded image blobs in ordinary audit/model context. Clear images on session/world changes and report unavailable/offline state honestly. Provide a repeatable agent-friendly capture recipe and live verify on the VM without owner assistance. Keep procedure design PR #92 unmerged and continue M2/M3 after this slice.

## Acceptance criteria

- [x] Rendered camera JPEG capture defaults to five seconds and never starts more than once per second, with bounded dimensions/bytes and no accumulating queue
- [x] Authenticated latest-image API and panel show capture time, scope and task; stale session images do not appear current
- [x] Runtime enable/interval/quality/size controls validate bounds; image blobs stay outside routine audit and model context
- [x] Core, protocol, session, panel and live VM checks verify capture and debugging retrieval with no owner setup
- [x] Users and agent tools export the actual panel map renderer with matching scoped data; read-only standalone rendering works without an open panel

## Implementation plan

Add a bounded screenshot message and shared capture config; Unity end-of-frame GPU downscale/readback/JPEG with a hard capture scheduler; latest-only low-priority bridge slot; strip image bytes from audit; validate and scope latest media in session; authenticated JPEG/metadata endpoint and panel view; runtime controls and agent-friendly capture recipe. Verify scheduler/transport/validation/scope/auth plus live image on the VM, then land and continue M2/M3 route and gathering fixes.

## Notes

**agent:codex/renki-milestones** at 2026-10-30T18:44:50Z

Implementation and fixture checks pass: just check ci (.checks/20261030T184419Z-Hlr3s2.log). Actual panel-renderer fixture rendered through native HTML preview without console errors. Live VM verification remains open; no completion claimed before deployment.

**agent:codex/renki-milestones** at 2026-10-30T18:48:54Z

PR #94 merged and deployed as ec7908c. Live JPEG: 960x540, 58,348 bytes, 15.7 ms capture work; viewing it revealed a D3D vertical orientation error. Follow-up corrects the GPU resize orientation. Live browser map snapshot API returned matching scope and 98 terrain tiles, 26 structures, 7 remembered container snapshots. Standalone map exceeded the native HTML tool input limit; narrowing atlas coverage to the default viewport and compacting JSON before repeating. Logout cleared the panel camera while retained terrain remained available.

**agent:codex/renki-milestones** at 2026-10-30T18:56:49Z

PR #94 and orientation/scope fix PR #95 are merged. Installed db85ea9 verified upright 960x540 JPEG (56,247 bytes, 12.8 ms capture work); image .checks/camera-20261030T185145079776Z.jpg with matching metadata. Audit .checks/audit-window-V6xP73.jsonl records 39 camera frames: five-second default and approximately one-second fast cadence (clock-stamped captures vary by milliseconds), 12.5-18.7 ms capture work and no jpeg fields. Native scheduler tests enforce capture-start spacing. Runtime 0.5-second interval was rejected; capture-off immediately returned unavailable. Full cadence/settings evidence .checks/camera-live-cadence-20261030T185257Z.json. Restored enabled capture with five-second interval.

The live browser map snapshot API returned the actual canvas plus viewport, layers, matching world UID/connection and displayed objects. Standalone just vm map-snapshot rendered successfully through native HTML preview: 134 merged terrain tiles, 512 bounded remembered landmarks, biome/support/height and layer controls, no console errors. Bundle .checks/map-snapshot-20261030T185145123989Z/. Fixed live-atlas scope and limited payload now fits the native rendering tool. Browser offline export retained fogged terrain and correct saved-world metadata; the camera cleared after logout. Native capture works unattended; explicitly minimized-window rendering has not been tested and is not established by these checks. CI/review logs are recorded in PRs #94/#95. Visual evidence supports diagnosis and does not replace task receipts.

**agent:codex/renki-milestones** at 2026-10-30T19:13:27Z

done to in-progress: User reported missing panel image: CSP blocks blob URLs; earlier live checks established capture and API but did not verify actual browser image decoding.

**agent:codex/renki-milestones** at 2026-10-30T19:13:27Z

Reopened after user reported no displayed screenshot. Native capture and authenticated JPEG retrieval were verified, but browser rendering was not: CSP img-src omitted blob: despite the client using object URLs. The prior completion note overstates panel-image verification; it remains valid for API/JPEG/cadence/map checks. Correct image-only policy and visible load errors, then verify actual browser naturalWidth and rendered scene before closing again.

**agent:codex/renki-milestones** at 2026-10-30T19:18:28Z

PR #99 merged the image-only CSP correction and visible camera load-error handling. Live authenticated browser reproduced blocked blob images before the fix, then loaded them after deployment. Actual camera img decoded naturalWidth960/naturalHeight540, hidden=false, displayed619x348 at x650/y149 inside a1280x800 viewport. The browser reported a current13-16ms camera frame and no console errors. This corrects the earlier gap: real browser display is now verified, as well as native/API capture. Screenshot /home/sothr/.t3/userdata/browser-artifacts/browser-screenshot-100-101-221-77-mvvcsyye-7bbc6d95.png; matching native JPEG .checks/camera-20261030T191650040559Z.jpg. Logout cleared it normally. Users need to reload a document opened before deployment because CSP belongs to the loaded page. Broader Preact panel redesign is delegated separately on an isolated branch; it includes prominent camera state, bounded sections and scrollable controls. The earlier minimized-window limitation still stands.

## Summary

PRs #94/#95 implement bounded camera and shared actual-renderer map snapshots. PR #99 fixes CSP-blocked browser images. Native capture, API, cadence/settings, audit byte exclusion, scoped map rendering, actual authenticated browser JPEG decode/display and logout clearing are verified; latest note records artifacts and explicitly untested minimized rendering.
