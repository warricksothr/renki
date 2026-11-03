# Handoff

Handoff of 2026-11-03 from `agent:codex/home-survey-delivery`, on
`t3/map-home-location` (base 0411381). The owner moved home, estimates the base
is 50–80 m across with sleeping/storage about 50 m apart, and asked for
configurable familiarization. Their later instruction explicitly authorized
filing, grooming, promotion and completion while away, superseding the earlier
planning-only hold.

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
Following to a new location still does not change saved home.

Three tickets were filed, groomed, promoted and claimed by this session:

- TKT-01M6FVN1XG (Base areas: separate sleeping, storage and return places).
- TKT-01M6FV695N (Waypoints: refresh bot places around a selected area).
- TKT-01M6FV6E40 (Home survey: familiarize with an area and update its places).

The local survey depends on these concrete base/refresh slices and reuses the
existing host, rather than undertaking the general expedition backlog. M7's
M2/M3 gates and other agents' claims remain unchanged. Ticket descriptions and
plans record this decision; live criteria must be earned independently.

## Validation and runtime

Focused home/atlas/behavior/goal/chat/app/protocol checks, the real plugin DLL
build and native bounded-travel tests have passed. An integration test covers
queued owner takeover and native stop of a running survey. A panel fixture
checks saved scope versus unsaved radius edits on desktop and mobile. Logs are
retained in `.checks/`; final gate/review/deployment outcomes belong in the
newest ticket notes and will be recorded at delivery.

The initial read-only VM observation at 06:43 UTC found Tas connected at the
game menu, without a current world/position. The session acquired the VM lease
as `agent:codex/home-survey-delivery` for deployment/validation. Do not infer a
successful new-home pass from offline checks. No new sleeping/storage roles
were guessed from historical marker names. Final runtime state and any
remaining native evidence will be recorded below before delivery.

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
