# Handoff

Replace this page at each handoff; tickets retain decisions and evidence.
Current work of 2026-11-04 by `agent:codex/explore-night`.

## Current state and owner choices

Branch `t3/renki-portal-travel-startup`, PR179. Installed plugin/controller
**f254e95** includes the panel views/multi-food plans/viewer sharing, activity-aware
food and coherent bedtime-clock correction. Later evidence-only commits need no
redeployment. VM lease released; take it again before live changes.
Keep Tas and private video online. Loot and automatic logout remain disabled.
Do not take older ticket claims or promote unselected drafts.

Tas returned home through checked stair-foot/top approaches: final t-15 succeeded
at[-204.18,32.97,-467.58], receipt .checks/task-0scabckn.json. Final post-deployment status
.checks/panel-batch-final-status.json shows[-204.3,32.97,-467.57],38.5/38.5HP,
injurySeq0,one active BoarJerky,two carried. Health-cap decline is food decay;
the normal keeper remains responsible for eligible refresh. No injury occurred. Leather helmet/chest/legs and full flint axe remain
his equipment. Restored home.whenStuck=true after supervised travel. Saved new
home remains[-204.39,32.97,-467.43],radius60; that protected no-harvest boundary
was never enlarged for exploration. Bed1274003559:489 is the owned saved spawn;
actual sleep/respawn remains untested.

Owner selected:200m exploration radius then return; bedtime shout when another
player is online; Meadows/Black Forest meals(tiers1–2) for idle, reserving later
and unknown meals for work with emergency override. The latest request adds
multiple foods per household plan and separate operations/config/settings/views.
Use the existing private panel address for sharing. No public publication was
requested or performed.

## Bedtime and activity-aware food

TKT-01M6KZ8KGV2MQANWADECVXBSPP (Ask players about bedtime once per game night):
mode presence.bedtime=shout is persisted. Fresh native audience/time, scoped
instant say, durable world/character calendar/night high-water marks; reserve
before send so uncertainty cannot replay. No automatic sleep/body interruption.
The complete audit19:24:00Z..19:45:00Z(.checks/audit-window-nxUeVD.jsonl) exposed
TWO shouts in the same night, t-13 at19:26:21Z and t-20 at19:30:50Z. This corrects
the earlier interim no-repeat inference. GetDay uses raw network time while
GetDayFraction is smoothed lighting, still0.998 when day advanced213→214.

The installed fix samples day/calendarFraction coherently from one native clock
reading and configured day length. Invitations require this optional raw field
and both raw/native night; older/missing clocks defer. Exact mismatch, restart,
midnight, rollback, missing/stale audience, uncertain send, persistence failure
and time-skip regressions are covered. Retain existing high-water marks; the
previous duplicate can conservatively suppress the next evening. Corrected
live midnight proof remains unearned; do not reset history or claim it passed.

TKT-01M6KZZ5F5C3YYJWX7RV2C3TMS (Choose food by activity and reserve better meals
for work): installed08590de,food.activityAware=true,food.idleMaxTier=2. Pure shared
explained decisions cover background maintenance and food goals. Tier overrides
are configurable; unknown/modded food is reserved while idle. Idle prefers low
tier and late refresh; work ranks observed health+stamina; no meal or low health
permits emergency food. Native eligibility, task ownership, freshness and receipts
remain authoritative. Explicit owner eat orders retain their item. Policy/audit
and runtime settings were verified live with carried BoarJerky/DeerStew; premium
versus cheap consumption was not tested because that choice was absent.

Prior gates:4e57efd fullCI220s;08590de fullCI176s including actual plugin build;
both committed reviews clean. Logs retained in .checks/20261104T19*.log.

## Deployed panel and validation

TKT-01M6M15PSGDEV44W48XCKXK1M5 (Group household food targets and split panel
configuration views):1–8 distinct supported foods with independent target/minimum
maps in one plan. Existing backend supports this; saved cards display the whole
mix. Forage/basic cooking only; do not imply unsupported cauldron recipes.

Operator views:Operations,Goals & kit,Places & base,People,Decisions,Settings.
They stay mounted to preserve edits and one video/map instance. Settings manages
a separate persisted random viewer credential. /view reuses map/video in monitor
mode without control components. Fragment exchange scrubs the link into a separate
HttpOnly/SameSite cookie. Server allowlist permits only filtered observation/map/
atlas/video playback; no chest contents,inventory,settings,audit/chat,commands.
Streams recheck revocation; established WebRTC media can continue until disconnect.
Viewer map positions and visible game content remain shared, not redacted broadcast.

Sharing is disabled after the live check, matching its initial state. just check vm viewer-check temporarily enables/rotates
only if initially disabled, restores that state, preserves existing links and
writes credential-free .checks/viewer-live.json. New dataDir/panel-viewer.json
must survive deployment. Do not publish the operator token in viewer links/logs.

Focused backend,desktop/mobile multi-food,viewer and navigation checks pass.
Full just check ci passed203s (.checks/20261104T200146Z-Ge5pgb.log), including66/66
desktop/mobile scenarios,Go race checks,actual plugin build and1033 native tests.
Committed review08590de..HEAD was clean(.checks/20261104T200526Z-33HvVe.log).
just check vm install passed32s(preflight,checksums,receipt,durable data retained),
just check vm online passed62s and just check vm viewer-check passed14s.
Logs .checks/20261104T200545Z-LsrO6X.log,200617Z-H2Dax5.log,200719Z-U73Mqu.log.
Live viewer login204,allowed reads200,private/control refusals403,rotation401;
credential-free results .checks/viewer-live.json. Both live and saved chest
contents are excluded. Private /api/video reports gameplay. Food/presence/home
settings persisted. Native calendarFraction is present(day215),but corrected
live night/midnight delivery remains unearned. Final status/map/inventory files
are .checks/panel-batch-final-*.json; exact final audit
.checks/audit-window-1RhaJy.jsonl. No body tasks were issued by viewer testing.

Bedtime,food and panel tickets are done/released. The partial survey ticket is
blocked/released with all safe-outing/return evidence retained; this is not full
coverage. Leave Tas and private output online. Before another live operation,
read current state/food and take a new lease; do not infer current stock from cache.

## Partial exploration and navigation follow-up

TKT-01M6KZ8KJGYWKA20FYWR6EP0HP (Survey safe land around saved home and record
map gains) is blocked after a safe partial survey. Within the requested200m
circle:3288→3400 canonical measured samples,112 new,1425 refreshed,0 lost;
395 landmarks unchanged. Evidence .checks/explore-night-atlas-{before,final}.json,
.checks/explore-night-final-summary.json. Do not claim full200m coverage.

Both strict dry building exits had routing failures. Existing bounded
supported_wading_v1 east-gate exit reached dry terrain[-144.85,31.51,-479.64]
without swimming/injury. Subsequent native200m scoped movement refused a fresh
Greydwarf/Elite exclusion and wet detours. A daylight retry reached southern
staging[-156.57,30.87,-483.29] but a roaming boar constrained onward routing.
No harvesting,combat,portals,swimming or injury. Return succeeded with separate
checked legs: east door[-156.8,30.97,-474.8] → warehouse west
[-169.69,31.03,-467.8] → stair foot[-189.51,31.26,-471.72] →
stair top[-193.01,32.97,-471.11] → saved home. Generic angled approaches can
still snag corners/fall toward the lower floor; do not repeat unchanged failures.

TKT-01M6M0HDCE27H971Q6SN08FHQN (Prefer clear building interiors and centered
stair approaches) remains draft. Current Walker follows navmesh corners or0.5m
supported local grid; no deliberate piece-center or stair-foot obligation.
Clearance preference plus doorway/stair connectors is proposed, not implemented.

## Architecture review and retained base evidence

Owner-authorized read-only architecture review is complete at docs/BEHAVIOUR-REVIEW.md.
Keep Runner/Template,durable goal adapters,leases/native authority; no behavior-tree
rewrite. Prefer explained pure decisions,scoped durable audits,immutable observer
snapshots and semantic replay. Food improves workload reads locally, but existing
Engine.List→view and Loot.Inspect→Observe can mutate tracker history on status reads.
TKT-01M6M0JWD09GYPTKV2MR28NW9N (Unify behavior decision audits and immutable
inspection snapshots) remains draft; do not promote without owner selection.

Prior harvest protection is deployed: saved base circles protect wood/stone/ore,
dry measured roots are required,underwater markers pruned,and static FirTree_oldLog
is a log. No destructive camp harvesting was done. Owner places/boundaries retained.

Warehouse west forge1:15324[-165.67,30.31,-473.28],east workbench1:15887
[-158.49,30.30,-476.39]. Close standing[-159,30.97,-475.07] repaired leather to400/400;
ordinary approach can stop at2.004m3D and fail strict native reach. Draft repair
approach ticketTKT-01M6KX627 remains unselected. Station levels are not exposed.
Flint axe came from door chest1:15433; clothing from upper barrel1:15599. Lower/
upper storage access is proven, not every third layer;1:15414/1:15438 remain
previously unread. Cached contents are not guaranteed stock.

Hearth1274003559:2382[-203.66,30.37,-470.56] accepted exactly2 Wood through the
half-wall opening from[-205.54,30.26,-466.89],with pack decrement/fuel increase
receipts(.checks/task-v5io_9bo.json). Native add_to supports ordinary finite fire
fuel. This proves reach/refill, not smoke safety in all weather. Return upstairs
succeeded. Keep ordinary ward/reach/heat/support/threat guards intact.
