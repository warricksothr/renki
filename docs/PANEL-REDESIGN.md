# Preact control room review

The workspace puts observation before configuration: health/food/current work
and safety actions, then a generous shared map alongside the game camera and
a compact activity timeline. Inventory/goals, places/storage and preferences
remain separate cards below. Section links and document scrolling keep them
reachable; long editors and activity scroll independently. Mobile stacks the
workspace and provides direct Map/Camera links and a fixed safety dock. A muted green palette and
quiet borders separate primary observations from supporting detail.

The migration deliberately retains the established control/API adapter.
`panel/src/workspace.jsx` mounts stable containers once; `controller.jsx` owns
those containers and the existing canvas renderer. Camera and timeline records
are Preact components, with a small shared status subscription for food and
persistent-goal context. This avoids rebuilding every input each second and
keeps focused settings edits intact. The adapter can be migrated further in
small changes without introducing another map implementation.

| Capability | Location / preservation |
| --- | --- |
| Stop, home, character logout | Persistent desktop safety bar; confirmation and original authenticated handlers retained |
| Panel session logout | Separate Sign out form |
| Health, stamina, food, current task/goal, overload/home | Overview; interrupted status clears current observations and disables actions |
| Map, entities, players, walls, paths, danger, terrain, hover | Same canvas renderer and data model, including grouped chest contents |
| Layers, pan, zoom, follow, exact export | Same `window.renkiSnapshots.map({maxWidth,quality})`; keyboard arrows/+/−/C added |
| Saved/offline atlas and world selection | Same scoped requests and fog; offline actors clear; rejoining restores follow mode |
| Camera | Prominent actual JPEG with scope checks, receipt/capture ages, decode errors and stale overlay |
| Camera settings | Enable and interval controls beside image; width/quality/pin/reset in full Settings |
| Inventory | Original game grid, wear/stacks/equipped/weight; focusable item descriptions; scoped snapshot on entry |
| Chests, places, zones, nearby and travel | Existing map hovers and complete editor in Places & storage |
| Goals and behaviour commands | Existing goal list/actions and owner chat/control APIs retained |
| Players, roles, marker sharing, Marco/Polo | Players & presence disclosure in Preferences |
| Voice and every runtime setting | Original model picker/settings adapter, including provenance, reset and pins |
| Owner chat | Say form beside Activity, using original owner authority |
| Task failures/history/diagnostics | Readable task reasons; expandable original lines; task modal, categories, pause, day picker and earlier pages retained |
| Standalone snapshots/previews | Built production assets embedded in self-contained read-only HTML; no separate renderer |

`just check ci` passes, including locked frontend reproducibility, camera state
and race tests, controller race tests, terrain maths and plugin interop. The
plugin DLL is skipped by the existing gate when no Valheim assemblies are
configured. No game actions, deployment or live VM checks were performed for
this redesign.

Native browser fixture checks passed at 1440×900 and 390×844 with 33 assertions:
all layers/export viewport, keyboard handlers, task dialog, historical paging,
settings focus/scroll, inventory, actual JPEG decoding, disabled/pending/error/
broken-image/stale/offline transitions, saved fog/rejoin and mutation refusal.
The recorded map bundle renders 134 terrain tiles and its grouped chest hover
retains observed contents/unknown distinctions. JPEG `naturalWidth` is 960 and
`naturalHeight` is 540. These recorded assets and replayed timing are fixture
evidence, not a new live game outcome. `html_preview` also rendered the complete
self-contained workspace with the image policy intact.

Build measurements against main cf59e5d (terrain renderer asset unchanged):

| Embedded asset | Before | After |
| --- | ---: | ---: |
| JavaScript | 75,230 B | 83,706 B |
| CSS | 13,050 B | 32,147 B |
| HTML shell | 11,484 B | 469 B |
| Total | 99,764 B | 116,322 B |

Total raw asset growth is 16,558 bytes. Gzip measurement is 29,396 → 35,741
bytes across these files; this is a size comparison, not a claim that the
controller negotiates compression. Preact is bundled locally and no CDN is
required. Node.js 20+ and npm are now build/CI prerequisites. Generated embedded
assets remain committed; `just panel measure` records these comparisons; `just panel verify` detects source/output drift.

Remaining review is aesthetic approval and optional owner-run live acceptance
at normal device sizes, using the camera already verified by the parent.
This branch must remain a draft PR until that review; it does not merge,
deploy or alter the owner's live VM.
