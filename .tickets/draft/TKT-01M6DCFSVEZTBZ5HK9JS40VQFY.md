---
schema: 4
id: TKT-01M6DCFSVEZTBZ5HK9JS40VQFY
title: "Panel: video gives up in background tabs; inspection poll hangs"
type: bug
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - tooling
assignees: []
milestone: M4
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:42:34Z
updated_at: 2026-11-02T15:56:49Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

Two panel polling bugs.

### 1. Live video gives up after the tab sits in the background

`panel/src/video.js:73-79` and `panel/src/video.jsx:31-34`. `transportFrame` is refreshed only by `requestVideoFrameCallback`, which browsers do not fire for hidden tabs. `tick()` keeps running and calls `failed()` after 5 s without a frame. After about a minute in the background, `attempts` reaches 5 and the operator comes back to "Video is unavailable".

Fix: skip the staleness check while `document.hidden`, and reset the attempts on `visibilitychange`.

### 2. A hung inspection request stops inspection polling

`panel/src/inspection.js:43-70`: `poll()` has no timeout, and `pending` blocks every later poll until the status scope changes. `camera.js` uses 8 s and video 2 s.

Fix: abort after 8 s. Add a Playwright or unit test for each.

### Goal

Goal 4.6 (review closed).

### Files

- `panel/src/video.js` (:73-79), `panel/src/video.jsx` (:31-34), `panel/test/video.test.js`
- `panel/src/inspection.js` (:43-70), `panel/test/inspection.test.js`
- `panel/src/camera.js` for the existing 8 s timeout pattern

### Verification

- `just check panel test` with a test per bug: hidden-tab frames do not count as failures, and a hung inspection request aborts after 8 s and the next poll runs.
- `just check panel ci` and `just check ci` (the controller embeds the built assets).

## Acceptance criteria

- [ ] Video does not exhaust its retries while the tab is hidden
- [ ] Inspection polls time out and recover
