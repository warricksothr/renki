# Repeatable panel UI review

The browser suite builds the actual Preact panel and embeds it with the existing
`scripts/map-preview.py` fixture transport. It uses the production shared map
renderer, not a second renderer or old screenshot HTML. Its committed inputs in
`panel/fixtures/` come from PR #102's recorded review: 134 terrain tiles, saved
atlas, scoped status/goals/chest observations and a real 960×540 JPEG. Inventory,
activity, settings and camera ages/expiry are simulated. Every screenshot says
**FIXTURE · NOT LIVE**. These are UI examples, not evidence of a new game outcome.

## Setup and runs

Prerequisites: Node 20+, npm, Python 3, just, and Chromium's OS libraries.
`just panel ui-install` installs npm's locked dependencies and Playwright's exact
Chromium revision. It does not install OS packages. If Chromium reports missing
libraries, an operator can install them using `just panel ui-install-system`.
Missing browser/font/data, launch failures and occupied ports are errors with
nonzero exit status; there is no browser skip that can make CI green.

- `just panel fixture` serves the recorded fixture on loopback port **18791**;
  `just panel fixture 18801` chooses another port. Stop with Ctrl-C. Use a new
  native T3 preview tab for interactive inspection; it remains the preferred
  interactive browser. In that fixture only, `renkiPreview.check()` reuses the
  incoming PR's regression checks, `renkiPreview.setCamera('stale')` selects a
  camera phase, and `renkiFixtureAdvance(31000)` advances fixture time.
- `just check panel ui-test` runs all named scenarios at desktop 1440×900 and
  mobile 390×844, retaining `.checks/panel-ui/`. This is also part of
  `just check ci`, via the panel's shared `ci` verb.
- `just check panel ui-test --scenario camera-stale --viewport mobile --output ../.checks/stale-review`
  narrows work. Comma-separated scenario names are supported. Output paths are
  relative to `panel/`, where its just module runs. `--port 0` chooses an
  available loopback port; the suite otherwise owns **18792**, never the live
  panel port. It starts/stops its own fixture server and fresh browser contexts.
- `just check panel examples` runs the same assertions for a curated feature
  set and writes `.checks/panel-examples/`, including `contact-sheet.html` and
  `contact-sheet.png`. `--viewport`, `--scenario`, `--output` and `--port` also
  work here. HTML contact sheets link individual full-size PNGs and remain
  portable alongside them; they are review artifacts, never production assets.
- `just panel ui-report` prints compact results and screenshot/diagnostic paths.
  `just panel ui-report ../.checks/panel-ui/result.json chest-hover` selects one
  scenario. Retrieve a specific PNG using its path; do not dump fixture HTML,
  JPEG data URLs, full accessibility trees or entire logs into agent context.

Choose a new explicit output directory when keeping multiple revisions or
repeated runs. Default directories are refreshed by later runs; `result.json`
and the contact sheet enumerate only the current run's artifacts. Use
`just check` for bounded terminal output and retained complete logs.

## Stable scenarios

| Name | Behavior and evidence |
| --- | --- |
| `overview` | Actual JPEG decode and dimensions, recorded terrain count, interval validation, top-of-panel example |
| `shared-regression` | Reuses PR #102's 35 fixture assertions: layers, keyboard, history, task failures, settings scroll/focus, inventory, camera phases, offline/rejoin and action refusal |
| `map-layers-export` | All layer controls update ARIA and shared export state; actual button downloads JPEG + matching JSON, exported JPEG decodes |
| `offline-fog` | Saved atlas with observation ages/fog, no live actors/image/inventory, safety actions disabled |
| `camera-no-image`, `camera-disabled`, `camera-pending` | Explicit no-scene states, image removed |
| `camera-failed`, `camera-broken`, `camera-stale` | API failure, invalid JPEG decoding, stale overlay/age with decoded image |
| `camera-temporary-expiry` | Effective 1s/base 5s, base editor and 30s expiry; simulated server expiry restores base with no write |
| `timeline-history` | Human-readable failed task, raw details collapsed, real task dialog/Escape, day paging, traces |
| `goals-inventory` | Recorded goal recovery/history/resume refusal, 32-slot inventory, focusable item description |
| `chest-hover` | Real pointer hover at export-derived coordinates: recorded contents, empty chest, unknown contents; individual screenshots |
| `controls-keyboard` | Real map keys, scoped waypoint submit and zero-radius default, player profiles, settings pin refusal/focus/scroll |

Each scenario also checks horizontal document overflow. Fixture writes are
always refused locally with 405; checks inspect the UI's refusal and request
scope/action header. They do not exercise a real authenticated mutation or
change any game state. Camera responses replay the current `config`,
`baseConfig`, and `override {id, config, expiresAt, connection, worldUid}` contract.
No suite step contacts a VM, game, credentials store or owner staging service.
Unexpected browser network traffic is blocked and reported as a failure.

## Reading results and reviewing visuals

`result.json` records status, named behavior checks, failure examples, separate
runtime/network/console error lists, screenshot paths and diagnostic paths.
Each `*.diagnostics.json` retains full assertion stacks, console messages,
fixture request bodies/headers and compact browser state. Failure screenshots
are attempted even after an assertion fails. Terminal failures are bounded;
inspect the named diagnostic rather than rerunning an unchanged passing suite.
The 503 camera response and invalid JPEG are deliberately supplied through the
in-document fixture; an unexpected real network or console error is still a
failure. An empty/missing suite is never a pass.

Time is fixed at 2026-10-30T19:00:00Z, Math.random is seeded (102), locale/timezone
are en-US/UTC, scale is 1, animations/transitions/carets are disabled and font
readiness is awaited. A locked local Inter Latin font is loaded only by the
fixture. Metadata records Node/OS, Playwright/Chromium versions/revision,
commit/dirty state and SHA-256 hashes of assets, font and inputs. Screenshots
from different OS graphics stacks can still differ. This suite deliberately
uses semantic assertions and reviewable images, with no broad screenshot
threshold or automatic pixel approval.

Start with `visual-review.json`, which is always **unreviewed**, and inspect the
individual images for each viewport:

1. Is hierarchy clear and text legible? Are fixture labels and stale/offline
   scope obvious?
2. Are controls, tooltips, errors, dialogs and mobile safety buttons visible
   without overlap or clipped content? Can focus/scroll reach long editors?
3. Do map layers/fog, chest provenance, camera decoding/freshness and temporary
   expiry match their labels and behavior?
4. Do timeline failures/history, goals, inventory and control errors explain
   what happened without exposing routine raw diagnostics?
5. Is each feature example useful and representative? Record concrete concerns
   with screenshot paths, expected behavior and severity.

Fill reviewer, decisions and `visualConcerns` in a **copy** of that report;
retain it with the artifacts. Semantic success, a model score and pixel
similarity do not prove visual correctness. There is no baseline auto-update
command. If a team adopts approved baselines, preserve this run's images,
metadata and signed-off report in a reviewed change; replace them only through
an explicit review of old/new images and documented rationale.

These checks cover fixture behavior in locked desktop Chromium and two viewport
sizes. They do not prove other browser engines, native mobile behavior,
production authentication, real capture timing, lease persistence/displacement,
VM rendering or game outcomes. Existing unit/controller tests retain scope and
race coverage; live acceptance remains separate and owner controlled.

## Video scenarios

`video-standby`, `video-no-source`, `video-connected`, `video-reconnect` and
`video-scope` run at both viewport sizes. These use a fixture peer with a locally
animated canvas stream, so actual browser video decoding/freshness gates are
exercised without a network relay. They prove no JPEG requests with diagnostic
details closed, safe covering, stalled-video recovery and logout preserving the
media session. Existing JPEG scenarios explicitly open diagnostic details first.
Screenshots remain **FIXTURE / NOT LIVE**; these cannot prove OBS game capture.
For real local media transport, use `just check streaming proof` as documented in
[LOCAL-VIDEO.md](LOCAL-VIDEO.md); it separately retains relay/FFmpeg/browser evidence.
