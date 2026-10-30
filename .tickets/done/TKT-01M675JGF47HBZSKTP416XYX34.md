---
schema: 4
id: TKT-01M675JGF47HBZSKTP416XYX34
title: Repeatable Preact fixture browser tests and feature examples
type: task
status: done
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
claim: null
archive: null
created_at: 2026-10-30T19:46:16Z
updated_at: 2026-10-30T20:00:25Z
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
- [x] Full just check ci and committed just check review pass; stacked draft PR targets PR102 branch

## Implementation plan

Extend PR102's read-only preview transport with checked-in recorded map/status/atlas/JPEG assets and a deterministic suite mode. Lock Playwright Chromium and a local font; run the built production UI with network denied outside an owned fixture server. Reuse preview-checks, then add named scenario behavior checks and desktop/mobile screenshots, exports, per-scenario diagnostics and feature contact sheets. Integrate the browser suite in panel ci, provide explicit prerequisite install and artifact inspection recipes, document human visual review and baseline limitations, then run full CI and committed review before publishing a stacked draft only.

## Notes

**agent:codex/preact-fixture-suite** at 2026-10-30T19:55:18Z

Implemented the fixture suite on distinct branch t3/preact-fixture-suite, fast-forwarded from owner main-derived checkout to PR102 commit9517fed before editing. Extracted recorded assets losslessly from PR102's saved preview:134 map tiles, atlas, status/goals/chests and960x540JPEG. Added locked Playwright1.56.1/Chromium141.0.7390.37 and Inter font, deterministic clock/seed/locale, strict offline fixture transport, named desktop/mobile scenarios, downloads, screenshots, compact results, categorized diagnostics and visual-review template/contact sheet. Native T3 inspection uses our own tab at127.0.0.1:18791; headless suite owns18792.

First complete suite passed30/30 combinations, reusing PR102's35assertions at both sizes. Browser interactions exposed a small production reset bug: Add place cleared the required radius instead of defaulting to0. Fixed and covered by real scoped submit/refusal. Fixture self-check originally raced layer tests; suite mode now leaves interactions to automation. No game/VM requests, credentials, staging, deployment or unrelated navigation/research work. Current full CI includes required browser checks; committed review/publication remain.

**agent:codex/preact-fixture-suite** at 2026-10-30T20:00:05Z

Published stacked draft PR #104, Add repeatable Preact browser tests and fixture examples: https://github.com/warricksothr/renki/pull/104. Registered with T3 immediately; base remains draft PR #102's branch t3/redesign-preact-control-panel at 9517fed. Implementation commit cd5b52c on t3/preact-fixture-suite. No merges, deployment, live game/VM requests, credentials, owner staging or unrelated native/research changes.

Final code gate just check ci passed (.checks/20261030T195620Z-zhi025.log, 87 seconds): 30/30 browser scenario/viewport cases, 76 grouped semantic checks plus PR102's 35 individual regression assertions at each viewport, 42 PNGs and zero runtime/network/console errors. Existing plugin DLL build remains skipped without Valheim assemblies; 362 plugin core/interop tests passed. Committed just check review 9517fed..HEAD passed without findings (.checks/20261030T195842Z-nR0e68.log). just check panel examples passed 18/18 feature cases with 30 PNGs (.checks/20261030T195756Z-ZVpXHs.log). Separate suite/example desktop and mobile overview PNGs matched byte-for-byte on this machine; this is repeatability evidence only, not visual approval.

Artifacts: .checks/panel-ui/result.json, .checks/panel-ui/*.diagnostics.json; .checks/panel-examples/contact-sheet.html and contact-sheet.png, individual feature PNGs and visual-review.json. Missing-browser probe deliberately failed with exit 1 and actionable setup instructions; failed result retained at .checks/panel-ui-missing-browser/result.json. Native T3 inspection used our own tab and fixture port 18791; automation owned 18792. Interactive/contact-sheet samples were inspected; visual templates remain unreviewed for explicit human approval.

Limits: recorded map/JPEG and simulated events/timing do not establish live capture/lease persistence/displacement, auth or game outcomes, other engines or native mobile behavior. No broad screenshot threshold or automatic baseline approval. docs/PANEL-UI-TESTING.md documents prerequisites, recipe/selector usage, artifacts, scenarios and visual checklist. Scope is complete; final ticket closure is the only follow-up commit.

## Summary

Implemented repeatable Preact UI automation and fixture examples in stacked draft PR #104: https://github.com/warricksothr/renki/pull/104, branch t3/preact-fixture-suite, implementation cd5b52c, targeting PR102 branch t3/redesign-preact-control-panel at 9517fed. Locked browser/font and deterministic recorded fixtures exercise the built UI/shared renderer in 30 desktop/mobile cases, preserve categorized diagnostics and screenshots, and produce labeled examples/contact sheets plus an explicit visual-review template. Fixed the zero-radius waypoint reset exposed by browser checks. Full CI and committed review pass; missing-browser failures are explicit. Docs/recipes and artifact/evidence paths are recorded in the final note. PR remains draft and unmerged/undeployed; live auth/capture timing/game outcomes remain outside fixture evidence.
