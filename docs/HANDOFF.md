# Handoff

Handoff of 2026-11-03 from `agent:codex/home-survey-delivery`, on
`t3/map-home-location` (base 0411381). The owner moved home, estimates the base
is 50–80 m across with sleeping/storage about 50 m apart, and asked for
configurable familiarization. Their later instruction explicitly authorized
filing, grooming, promotion and completion while away, superseding the earlier
planning-only hold.

## Continuation: old-house return and shallow water

The owner returned Tas to the old house and explicitly requested self-navigation
to the saved new home and a new bed. The initial 14:35 UTC observation showed
sheltered Tas 504 m from home, empty inventory, 25 HP and no meals. Under the VM
lease, normal withdrawals supplied three yellow mushrooms, two neck tails, two
wood and one rag tunic. Tas equipped the tunic, ate automatically and cooked both
tails with native output receipts. Travel began at full meal-adjusted health,
with two mushrooms and one cooked tail in reserve. InjurySeq remained 16702.

A direct dry/hostile-avoiding return from the cooking position refused without
movement. A checked intermediate walk to the previously observed exterior
landing succeeded (t-65, `.checks/task-jolb8tsa.json`). The next native return
(t-66) made southbound progress, then refused bounded dry bank departure near
[-47.33,30.95,-117.88], reference water30, observed footing approximately30.9.
Evidence `.checks/task-mbs1nked.json` and `.checks/new-home-bank.json`. It did not
reach the new home and no new bed has been claimed. The latest pre-change
observation was grounded near[-47.4,30.64,-118.57], full56.3HP, no new injury.

The owner watched the shallow-ground refusal and requested a distinction between
safe wading and swimming. TKT-01M6GXWKM2SC3K27C1JE98Z0AV (Travel: allow checked
shallow-water wading without swimming) owns the explicit `move_to` option
`wade:true`, alongside `dry:true` and `avoidHostiles:true`. It preserves strict
dry household/survey behavior. See BRIDGE-PROTOCOL.md and TESTING.md for mode,
threshold reserves and validation. Build98481da passed the full gate (214s), committed review and checksum-verified
full deployment (`.checks/20261103T145310Z-e3iTza.log`). Tas rejoined grounded in
0.3m of water at[-46.95,29.74,-120.57], full54.4HP with two active foods.
The first wading return t-1 walked about12m south and reached dry ground near
[-52.96,30.64,-130.23] without observed injury, then falsely refused its live
normal check (`.checks/task-vgfo67v1.json`). Installed Character source shows
GetLastGroundNormal returns the transient m_groundContactNormal accumulator,
which UpdateGroundContact clears to zero; it is not a retained support normal.
The correction validates the actual footing ray's walkable normal and preserves
the original slope limit; body contact flags remain required. A regression uses
the recorded body/footing with explicitly synthetic ray normals. Corrected
build3eb180f passed just check ci (166s), committed review and full deployment.
Its native return travelled roughly350m farther toward home without observed
injury, stopping57m short at[-188.95,29.49,-412.48]. Actual footing29.45 and
local water30.1009426 put depth less than1mm past the planner's0.65m allowance,
while still grounded and below the actual0.8m allowance. Evidence
`.checks/task-h56xv1tf.json`, `.checks/new-home-corrected-3.json`.

A follow-up distinguishes current actual depth from future planning headroom:
current supported body/footing may use the0.15m settling reserve, but the next
checked midpoint and endpoint must recover it. Recorded-wave regressions pass
in `.checks/20261103T150631Z-qgBi60.log`. Full gate/deployment and a new native
attempt remain pending. Latest15:08 observation is[-189.95,29.12,-412.8],
grounded in reported0.8m water, full38.8HP, injurySeq0, only166s of cooked-tail
meal left and no carried food. Recheck actual depth before retrying; idle drift
has deepened the water. Home and bed are still unconfirmed.

The owner also asked about atlas-generated chains and following player paths.
Current tooling has named waypoints, live-validated route trails and survey
checkpoints, but no general durable ordered itinerary/cursor/resume tool. The
atlas has elevation/biome/water at4m; ground-paint path surfaces are not captured.
A one-off retained-terrain candidate calculation is in
`.checks/new-home-atlas-route.json`: it leaves unknown/stale/deep samples out and
found only a partial connection. That is planning evidence, not proof of route
clearance or impossibility. Future work is filed as drafts: TKT-01M6GYMV97NYP025K9D4C0FRS7 (Routes: save waypoint chains
with durable execution progress) and TKT-01M6GYMVB10AP9WBAW7Y176PQC (Movement: observe ground
paths and prefer useful path segments). Neither is implemented or enabled by
this continuation. They require separate route-host/surface observation work.

## Home familiarization

Implemented a local `familiarize` profile in the existing durable goal host,
with configurable per-world named bases and distinct sleeping/storage/return
bindings. See [EXPLORATION.md](EXPLORATION.md#local-home-familiarization) for the
implemented controls, admission requirements, evidence and limitations.

Configure **Places & storage → Base area & familiarization**, save/select the
base, then request `Tas, familiarize yourself with home` (or `this area`, `the
area`, `survey home`). `survey status` reports progress; stop or any newer body
order wins. Radius is 10–250 m, total duration 2–20 minutes and total travel
50–4000 m, including return. Defaults are editable, not measurements of the new
base. `this area` means the saved base; temporary `here` anchors are deferred.

The profile uses bounded dry hostile-avoiding native `move_to`, records actual
3D travel/trails, checks post-receipt supported floor arrival, visits selected
sleeping/storage connections in both directions and reserves return budgets.
Unsafe conditions stop with a diagnostic. A restart with uncertain work cannot
replay it. Coverage separates measured/new/refreshed/unknown/stale 4 m terrain
samples from visited ground and observed objects. Partial results are expected;
no terrain result claims complete room, chest or floor inspection.

Scoped learning preserves owner edits, pinned roles, suppression, other bases
and old owner-managed `home storage`. It creates multiple evidenced places;
no chest is opened and no bed is claimed. Wood delivery uses selected storage;
marker rest uses sleeping with existing ownership/shelter/floor checks. Other
food/resupply routines retain local point constraints around return/home.
Selecting a large base does not migrate their definitions or teach portals.
Death-near-home/shunning and short-home heuristics also remain local checks
around the return point, not the whole base circle. Following to a new location
still does not change saved home.

Delivery is [PR #176](https://github.com/warricksothr/renki/pull/176), initial
implementation commit **0152cb0**, plus a follow-up that normalizes empty storage selections
so checkpoint serialization cannot spuriously invalidate an unassigned base.
An unassigned base still explores to learn candidates when terrain is already
refreshed; loaded terrain alone does not end that first learning pass.

Base areas TKT-01M6FVN1XG (Base areas: separate sleeping,
storage and return places) and refresh TKT-01M6FV695N (Waypoints: refresh bot
places around a selected area) are done and released. TKT-01M6FV6E40 (Home
survey: familiarize with an area and update its places) has its offline criteria
earned and remains in review for live survey/50 m building-route evidence.

The local survey depends on these concrete base/refresh slices and reuses the
existing host. M7's M2/M3 gates and other agents' claims remain unchanged. A
note on TKT-01M6EFQN74 (Atlas: first-seen time and a measured-coverage query)
identifies the shared schema/query to reuse; its separate coverage HTTP endpoint
is still draft work and was not claimed or implemented here.

## Validation and runtime

`just check ci` passed in `.checks/20261103T070445Z-99MVRa.log` (200 s): 60/60
desktop/mobile scenarios, controller race checks, actual plugin DLL build,
Core/interop and repository tooling. `just check review` passed with no findings
at 0152cb0 (`.checks/20261103T065700Z-y3SY0h.log`). Focused survey tests passed
three consecutive race runs (`.checks/20261103T065308Z-LNJQsO.log`). They cover
full/partial terrain, 50 m role separation, wrong floors, failed return, food
reserve, uncertainty/restart, edits, queued takeover and running native stop.
After the final controller changes, `just check controller test-race` passed
(`.checks/20261103T070843Z-ldtKcP.log`). Ten repeated start/stop/takeover race
runs passed (`.checks/20261103T070437Z-HAMvOv.log`).

`just check vm deploy` passed under the lease, verifying package checksums and
joining DizzysHellheim. The initial live controller reported **0152cb0**;
the follow-up uses a controller-only deployment, preserving the running game. Evidence:
`.checks/20261103T065712Z-kPMpX7.log`, with before/after observations and API
responses in `.checks/home-survey-live/`.

**Live survey could not start.** At 06:58–06:59 UTC Tas was swimming near
`[-167.7, 28.75, -454.4]`, at 25/25 HP and 50 stamina, with no active or carried
food. Saved home is already the owner's new point
`[-204.39, 32.97, -467.43]` (set by Nimah), about 39 m away. The previous
handoff's older home coordinates must not be reused.

Saved an editable **new home** base centered on that existing home: 60 m radius,
10 minutes, 800 m total travel, return bound to saved home. Sleeping/storage
remain unassigned; their approach coordinates were not guessed. Existing home
and waypoints were preserved. The authenticated start request refused on
health/footing/capacity/boundary checks **before creating any survey**; the
subsequent API still returned no surveys. Tas remained idle with unchanged HP
and stamina. No native walk, coverage gain, checked return or separated-building
proof is claimed.

**Later incident and final state supersede the initial water observation.** At
07:04:51 and 07:04:53 UTC a Greydwarf hit Tas for 13.27 and 13.56 at that
water position. Existing retreat dispatched plain flee within 0.2 s; Tas died
about two seconds later and respawned at the old bed
`[-31.08, 36.55, 15.19]`. Existing after-death home handling attempted a dry
return to the saved new home, then failed `unreachable` after local support/route
refusals. No familiarization goal had been created or dispatched. The incident
is recorded on TKT-01M62N256M (Plugin: flee task, and retreating when health is
low), without taking over its claim. It does not earn a survival or survey pass.

Full audit: `.checks/audit-window-NgxmRf.jsonl`; bounded report:
`.checks/20261103T071106Z-WhOy9o.log`. Follow-up **c088d87** passed committed
`just check review` with no findings (`.checks/20261103T070956Z-bsniPE.log`) and
`just check vm deploy-controller` (`.checks/20261103T071033Z-KhYdFf.log`), keeping
the game running. The final 07:11 UTC observation reports c088d87 idle on dry
ground near `[-46.37, 33.78, 10.15]`, 25/25 HP, 50 stamina and no active/carried
food. It is outside the saved new-home boundary. Evidence:
`.checks/20261103T071114Z-ECsbpC.log`.

The next live pass needs supplied, dry supported staging at the new home and
owner-marked sleeping/storage approaches. Confirm the editable boundary and
roles in Places, then start familiarization and retain coverage/waypoint diff,
each direction, floor arrival and return evidence. The current old-bed location
cannot start the saved new-home profile; neither water nor a distant respawn
should be forced into a mapping pass. Leave the two native criteria unticked.
The VM lease is released at handoff; read current lease status before further
live changes.

Atlas schema 2 transactionally migrates schema 1, preserving retained first
receipt/source for honest new-versus-refreshed accounting. Evicted/reinserted
cells do not preserve lifetime provenance. See [ATLAS.md](ATLAS.md).

## Other work retained from the prior handoff

The prior handoff reported b5cc019 deployed. PRs #168–174 supplied loot's anchor,
adaptive reach, VM lease, passive follow loans and perception-skew tolerance.
TKT-01M6EWN4WH (Behaviours: passive mode lends an owner's follow to passive
instances) still needs its owner-in-game acceptance check: settled owner fells a
tree, Tas collects loose drops, ignores a stack dropped at the owner's feet,
resumes follow when the owner leaves, and yields permanently to a newer order.
The previous notes observed successful passive runs before the skew fix and
churn that #174 addresses. Read that ticket's latest evidence before retesting.

TKT-01M6EF5DTG (VM: exclusive lease for deploys and live trials) has a separate
live A/B refusal check outstanding; its notes describe the refused request.
Neither of these was silently closed by the home-survey work.

Other drafts remain the owner's future queue: portal following
(TKT-01M6FT1ZVW), passive gather (TKT-01M6F877TX, depends on anchor activity
inference TKT-01M6F877Q8), loot anchor panel (TKT-01M6EYEMTX), needs ledger
(TKT-01M6F877XV), and agent-supplied passive lists (TKT-01M6F8780V). Portal
following may conflict with the separately claimed building/portal memory work
TKT-01M6AYP8; do not take it over implicitly.

Known unrelated gaps remain: passive re-lend churn when a second passive
instance preempts, a zero-health frame without a death can lose resumed follow,
custom settle shapes expire after ten minutes, and startup skips nine saved
unknown goal instances. Do not delete owner settings to silence those warnings.
The 32 old in-progress claims have not been taken over. App tests have a known
occasional load timeout (TKT-01M6ESNXKF); inspect retained diagnostics first.

## Operating rules

Read AGENTS.md and docs/TESTING.md before work; docs/VM.md before live changes.
Run repeated tasks through `just`, focused checks through `just check`, the
full gate as `just check ci`, and committed review as `just check review` before
pushing. Write ticket prose with title beside its first ID, record decisions and
remaining native checks, and identify every write with your actor/revision.

Take `just vm lease ACTOR` before VM mutation and `just vm release` afterward.
Do not force another holder's lease. Deployment uses `just check vm deploy`;
worktrees need gitignored `vm.local` and `plugin/game.local.props` copied without
printing them. Secrets and local settings must never be printed. Keep Tas
online after checks by default; an opt-in deployment restart is separate from
trial cleanup. Do not kill the VM's `renki-ctl` service. Verify merge ancestry
against fetched origin/main if a later session merges the PR.
