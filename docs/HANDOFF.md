# Handoff

Replace this page at each handoff; tickets retain decisions and evidence.
Handoff of 2026-11-03 from `agent:codex/portal-deploy`.

Read AGENTS.md, docs/ROADMAP.md, docs/BEHAVIOURS.md, docs/PORTALS.md,
docs/TESTING.md and docs/VM.md before resuming live work.

## Owner decisions and branch stack

The owner selected portal following, authorized its deployment, then selected
all three next drafts: deployment preflight, anchor activity tracking and Loot
panel visibility. The owner explicitly chose **keep the game offline** while
this batch is implemented. Do not start Valheim for these implementation checks.

- PR #176, `t3/map-home-location`, head dd0042f, holds the newer installed
  prerequisite: atlas schema 2, home surveys, checked wading, chest inspection,
  door cleanup and proactive/blocked-escape protection.
- PR #177, `t3/continue-renki-bot`, head a148089, is stacked on #176 and adds
  portal following. The installed runtime from this branch is **871d818**;
  a148089 adds deployment records only.
- The next batch is on `t3/renki-runtime-loot-batch`, based on a148089. Its PR
  should be stacked on `t3/continue-renki-bot`. No PR was merged in this session.

## Current implementation batch

- TKT-01M6H80F6B (VM: detect runtime drift before installing older builds):
  both install paths stage and preflight before stopping/replacing the current
  runtime. The manifest records committed source/branch/ancestry, supported atlas
  schema/tasks, checksums and actor. The staged controller reads config, saved
  task permissions and SQLite schema without startup/migration/writable stores.
  Unknown/newer/unmerged builds, changed checksums and incompatible data refuse.
  An identifiable clean legacy ancestor can be adopted. Dirty/uncommitted source
  is refused. Successful installs retain a receipt with verified hashes and prior
  game/controller state. Installation keeps the game offline; only deploy/start
  explicitly starts it. `just check vm deployment-refusal` exercises compatible
  read-only inspection and both actual refusal entry points while retaining the
  running panel, controller PID and installed/config/data hashes. Run only with
  the game already offline, under the VM lease, on a committed reviewed candidate.
- TKT-01M6F877Q8 (Behaviours: infer the anchor's activity from changes around
  them): a pure tracker infers felling/mining/picking from vanished resources
  within 12 m of a live authorized anchor, carries prefab/ID/position/time/age,
  and expires after 30 s. Mixed evidence chooses nearest, then kind and ID.
  Duplicate/old/stale/gapped frames, changed scope, jumps, capped kinds and scan
  edges cannot invent new activity. `View.Activity` and behaviour status expose
  it without enabling gathering or dispatching tasks. These are inferences;
  current perception cannot identify the actor or prove every scan complete.
- TKT-01M6EYEMTX (Panel: show loot's anchor, settle area, reach and drop
  decisions): `/api/behaviours` carries per-instance scoped, timestamped Loot
  diagnostics. Selection and diagnostics share the decision function. A read-only
  card shows enablement, anchor/settle/reach, recent inferred work, all effective
  parameters/defaults and drop reasons. The map's Loot reach layer shows the
  anchor, settle/reach circles, eligible filled drops, deliberate crosses and
  hollow out-of-reach drops. Select an instance if several exist. Offline,
  stale, changed-scope and saved-atlas views clear the live overlay; polling the
  same frame cannot renew it. Disabled Loot stays disabled. The owner-in-game
  reach-over-time criterion remains unearned.

The full `just check ci` gate passed (225 s), retained in
.checks/20261103T180530Z-ShHgUG.log. Focused Go/race checks and the desktop/mobile
`loot-reach` UI scenario passed.
The UI scenario also verifies distinct canvas colors, layer toggling, expiry of
an unchanged frame and no non-GET requests. Committed review and native game-offline refusal verification follow the first
implementation commit. The activity ticket is done and released; the panel
ticket is in review and released with its in-game criterion unchecked.

## Installed runtime and required offline state

The initial portal-only deployment of 77a527c installed with verified checksums
but its controller exited: `atlas schema 2 is newer than supported schema 1`.
Saved settings also included `inspect_container`. The actual baseline was
unmerged PR #176 (runtime 2e8c6d1), newer than the older main handoff suggested.
The failed deploy's local wait was interrupted, then `just vm stop` stopped the
game. No atlas/settings were deleted, reset or downgraded. Never deploy 77a527c
alone against this runtime.

Combined build **871d818** installed successfully with `just check vm install`
(.checks/20261103T174828Z-CbVn1x.log). Controller/Core/plugin checksums verified.
`just vm panel-start` started only the controller/panel. `/api/status` reported
871d818 and disconnected; VM status confirmed Valheim stopped and controller
PID 39640 running. `/api/behaviours` confirmed **Loot disabled and not running**.
The combined CI gate and committed review passed. Deployment lease released.
The new implementation batch has not been installed in production.

PR #176's historical emergency stop was around 16:12Z: position
[150.65,70.48,-33.81], 15.8/25 HP, no active/carried food and two targeting
Greylings. This is not a fresh health/location claim after the brief failed
startup. Grave recovery and a safe return remain unresolved; menu HP0 is not
proof of death. Saved home is [-204.39,32.97,-467.43], base radius 60 m, storage
approach [-169.13,30.97,-467.75]. Sleeping is unassigned; no free Tas bed was
confirmed or claimed. Keep Loot disabled: it previously entered a portal during
stuck recovery and stranded Tas. Retain #176's final guard that immediately
fails a body task when portal input is refused.

## Portal following and remaining native checks

TKT-01M6FT1ZVW (Follow: go through the portal the leader just used) stays in
review, released, with live criterion 5 unchecked. The plugin remembers nearby
leader/source observations for 8 s, even during passive pickup. Disappearance or
remote position jumps can start a bounded native crossing; returned follows
accept that evidence. Only the observed source and unchanged destination are
admitted, scoped to route planning and the active input step. Cancellation,
standing, failure and teleport revoke the allowance; other portals stay blocked.
Valheim's cargo/teleport rules remain authoritative. Cargo and crossing failures
explain why Tas stopped without silently walking home. Arrival clears geometry
and resumes follow. Offline policy/protocol/chat coverage does not prove entry
into a native portal or actual arrival.

After the owner stages a safe supervised rejoin:

1. Take `just vm lease agent:tool/session`; start a reviewed installed build.
2. Follow Nimah through a connected portal pair and back; retain departure,
   arrival/failure and task outcomes. Tick criterion 5 only after the pair passes.
3. Agree with the owner before enabling passive Loot. Check departure during a
   felling-drop pickup, native cargo refusal, stop/new-order takeover, and refusal
   of other portals. Do not enable Loot for an offline deploy.
4. Record Loot's anchor, settle and growing/reset reach while the owner stands,
   fells and leaves. Record the real reach over time for the panel ticket.
5. Release the VM lease after cleanup.

Also still owed:

- TKT-01M6EWN4WH (Behaviours: passive mode lends an owner's follow to passive
  instances), criterion 6: collect felling drops and fall back in; leave an
  owner-dropped stack; yield to movement mid-pickup and any new order. Earlier
  a892cfa checks worked but churned before #174; the full rerun is missing.
- TKT-01M6EF5DTG (VM: exclusive lease for deploys and live trials), criterion 1:
  A/B refusal check with `/api/act/home` as the refused POST. Existing claim stays.

## Next drafts and known gaps

Only the three selected drafts were promoted. Passive gathering
TKT-01M6F877TX (Behaviours: passive gather works alongside the anchor) still needs
the owner's selection and deliberate policy work: existing gathering is
beech-only, excludes players within 32 m of standing trees while passive reach
is 30 m, and must fix re-lend churn. It is not an enable switch. Later work is
TKT-01M6F877XV (Behaviours: needs ledger ranks passive work by what the base lacks)
and TKT-01M6F8780V (Agent: supply the passive list, with validation and built-in
fallback). Do not promote these yourself.

- The 32 older in-progress claims remain; do not take them over without the owner.
- A zero-health frame without a subsequent death can lose a lent follow.
- Non-default settle shapes are forgotten after ten minutes without being asked
  for, even while busy. Other perception checks still have older tolerances.
- Nine saved `goal/<hash>` instances warn as unknown; ask before deleting them.
- App timing/race flakes: TKT-01M6ESNXKF and TKT-01M6EWJ9. Read retained diagnostics
  before a focused rerun; do not relax production freshness to make tests pass.
- Ignored `vm.local` and `plugin/game.local.props` are configured. Never print
  their contents, panel/CPA tokens, provider keys or server passwords.
- Do not kill `renki-ctl` on the VM. Use the lease for mutations; read-only
  inspection needs none. Write local JSON without a BOM.
