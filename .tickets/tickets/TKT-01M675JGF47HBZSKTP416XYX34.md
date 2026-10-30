---
schema: 4
id: TKT-01M675JGF47HBZSKTP416XYX34
title: Repeatable Preact fixture browser tests and feature examples
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - tooling
  - docs
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/preact-fixture-suite
  branch: t3/preact-fixture-suite
  worktree: /home/sothr/.t3/worktrees/renki/t3-f79ce10e
  commit: 9517fedbd9b0bc123e988db7846e3133337a15b3
  session: null
  claimed_at: 2026-10-30T19:46:29Z
  expires_at: null
archive: null
created_at: 2026-10-30T19:46:16Z
updated_at: 2026-10-30T19:58:42Z
created_by:
  id: agent:codex/preact-fixture-suite
  name: ""
updated_by:
  id: agent:codex/preact-fixture-suite
  name: ""
extensions: {}
---

## Description

## Acceptance criteria

- [x] Deterministic fixture suite exercises built Preact UI and shared renderer at desktop and mobile sizes without live requests
- [x] Semantic camera, map, timeline, controls and accessibility checks retain categorized diagnostics and screenshots
- [x] Just recipes provide preview, browser setup/test and labeled feature examples with visual review guidance
- [ ] Full just check ci and committed just check review pass; stacked draft PR targets PR102 branch

## Implementation plan

Extend PR102's read-only preview transport with checked-in recorded map/status/atlas/JPEG assets and a deterministic suite mode. Lock Playwright Chromium and a local font; run the built production UI with network denied outside an owned fixture server. Reuse preview-checks, then add named scenario behavior checks and desktop/mobile screenshots, exports, per-scenario diagnostics and feature contact sheets. Integrate the browser suite in panel ci, provide explicit prerequisite install and artifact inspection recipes, document human visual review and baseline limitations, then run full CI and committed review before publishing a stacked draft only.

## Notes

**agent:codex/preact-fixture-suite** at 2026-10-30T19:55:18Z

Implemented the fixture suite on distinct branch t3/preact-fixture-suite, fast-forwarded from owner main-derived checkout to PR102 commit9517fed before editing. Extracted recorded assets losslessly from PR102's saved preview:134 map tiles, atlas, status/goals/chests and960x540JPEG. Added locked Playwright1.56.1/Chromium141.0.7390.37 and Inter font, deterministic clock/seed/locale, strict offline fixture transport, named desktop/mobile scenarios, downloads, screenshots, compact results, categorized diagnostics and visual-review template/contact sheet. Native T3 inspection uses our own tab at127.0.0.1:18791; headless suite owns18792.

First complete suite passed30/30 combinations, reusing PR102's35assertions at both sizes. Browser interactions exposed a small production reset bug: Add place cleared the required radius instead of defaulting to0. Fixed and covered by real scoped submit/refusal. Fixture self-check originally raced layer tests; suite mode now leaves interactions to automation. No game/VM requests, credentials, staging, deployment or unrelated navigation/research work. Current full CI includes required browser checks; committed review/publication remain.
