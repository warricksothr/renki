# Handoff

The state of the work at the last handoff, and what the next agent needs
that the tickets and the roadmap do not already say. Replace this page at
each handoff rather than appending to it.

Handoff of 2026-11-03, from `agent:codex/portal-deploy`. The owner selected
TKT-01M6FT1ZVW, **Follow: go through the portal the leader just used**, from
the previous handoff's drafts.

## Read first

1. AGENTS.md, for recipes, tickets and committed review.
2. docs/ROADMAP.md, especially working in parallel and the conflict hotspots.
3. docs/PORTALS.md, for portal following and the remaining general travel work.
4. docs/BEHAVIOURS.md, docs/TESTING.md and docs/VM.md before live work.

## Current change

Branch `t3/continue-renki-bot` combines portal following with the newer runtime
baseline from [PR #176](https://github.com/warricksothr/renki/pull/176), head
dd0042f (runtime 2e8c6d1). Portal delivery is
[PR #177](https://github.com/warricksothr/renki/pull/177). The ticket carries its
plan and remaining live evidence. Neither PR has been merged in this session.

- The plugin remembers a nearby player's connected portal for 8 s, including
  while a passive pickup owns the body. A disappearance or remote position
  jump can start a crossing. A returned follow can use that recent evidence
  instead of being rejected because its leader is already gone.
- A crossing admits only the observed source and its unchanged destination.
  Planning admission is scoped to the walker call; input admission resets on
  each physics step and on standing/cancellation. Every other portal stays
  blocked. The collision-cache ticket TKT-01M6AYP8 (Cache building collisions
  and remember portal links) retains its older claim; this change only adds
  the scoped exception to its guards.
- Valheim's trigger and teleport logic remain authoritative. Native cargo
  refusal is `portal_cargo`; other transition failures are `portal_failed`.
  Both explain the failure and ask the owner to return, without dispatching
  the usual lost-follow trip home.
- The transition bounds approach, trigger refusal, teleport and arrival
  separately. Arrival drops local geometry and resets follow movement.
- Offline coverage includes pure transition/departure policy, shared protocol
  fixtures, the prompt return of a passive loan after the owner disappears,
  and the chat explanation without going home. Native behavior is unproven.

## Runtime recovery and deployment

The first deploy of 77a527c installed successfully but the controller exited:
`atlas schema 2 is newer than supported schema 1`. Saved settings also included
`inspect_container`. The actual VM baseline was newer unmerged PR #176, rather
than b5cc019 as the original handoff assumed. The deploy wait was interrupted
and `just vm stop` stopped the game. No atlas or settings were deleted, reset or
downgraded. Do not retry 77a527c on its own against this runtime.

The newer handoff records why Tas had been offline: at about 16:12Z, position
[150.65,70.48,-33.81], 15.8/25 HP, no active/carried food and two targeting
Greylings, then emergency stop. These are historical observations, not a fresh
health/location claim after the failed deploy's brief startup. Grave recovery,
a safe return to the new home, and a free Tas bed remain unresolved. Menu HP0
is not proof of death. Saved new home is [-204.39,32.97,-467.43], with a
60 m base radius and storage approach [-169.13,30.97,-467.75]. Sleeping remains
unassigned; no bed was claimed. Keep **Loot disabled** as PR #176 left it: earlier Loot
entered a portal during stuck recovery and stranded Tas. PR #176's final guard
now fails the body task immediately on refusal; retain that fix with the scoped
portal-follow exception. Do not blindly rejoin the saved pursuit.

The deployment now integrates PR #176's atlas schema 2, base surveys, checked
wading, door cleanup, read-only chest search and proactive/blocked-escape
protection. Install the combined build with `just check vm install`, then
`just vm panel-start` so schema/settings compatibility can be checked without
starting the game. Record the actual verified version and final status in the
portal ticket. A safe supervised rejoin with the owner must precede field work.

For portal following, once Nimah can supervise a safe start:

1. Take `just vm lease agent:tool/session` and start the installed combined build.
2. Ask Tas to follow. Walk through a connected portal, wait on the far side
   for Tas to arrive and resume following, then do the same trip back.
3. After agreeing to re-enable passive Loot for the test, check a departure
   while Tas is picking up felling drops. The returned follow must take the
   source portal. Do not re-enable Loot simply to prepare an offline deploy.
4. Verify cargo refusal, stop/order takeover during approach and refusal of
   other portals. Retain `portal departure`, `portal arrived`, `portal failed`
   and task outcomes; tick the live criterion only after the pair trip passes.
5. Release the VM lease after cleanup. The game does not need to run for the
   next offline implementation batch.

The previous live checks also remain:

- TKT-01M6EWN4WH (Behaviours: passive mode lends an owner's follow to passive
  instances), criterion 6: on b5cc019 or later, follow a settled owner felling
  a tree, collect drops and fall back in; leave an owner-dropped stack alone;
  yield when they walk away mid-pickup; any new owner order wins. Earlier runs
  on a892cfa worked but churned until #174; the full rerun is still missing.
- TKT-01M6EF5DTG (VM: exclusive lease for deploys and live trials), criterion
  1: the live A/B refusal check from its notes, with `/api/act/home` as the
  refused POST. It remains in review under its existing claim.

## Drafts still awaiting the owner

Do not promote these without selection:

- TKT-01M6H80F6B (VM: detect runtime drift before installing older builds),
  filed after the deployment mismatch in this session. It stays in draft.

- TKT-01M6F877Q8 (Behaviours: infer the anchor's activity from changes around
  them), then TKT-01M6F877TX (Behaviours: passive gather works alongside the
  anchor). Gathering must fix the known re-lend churn before becoming the
  second passive instance.
- TKT-01M6EYEMTX (Panel: show loot's anchor, settle area, reach and drop
  decisions). Passive-run data is already in `/api/behaviours`.
- Later: TKT-01M6F877XV (Behaviours: needs ledger ranks passive work by what
  the base lacks) and TKT-01M6F8780V (Agent: supply the passive list, with
  validation and built-in fallback).

The owner's idle/passive decision is on epic TKT-01M62VXYQB (Behaviours:
templates and instances the bot runs on its own): idle work needs an idle
body; passive work can borrow a settled owner's follow and yields to movement
or any owner order. Loot and future gathering are passive and on by default in code; the
current runtime Loot setting is deliberately disabled and must be preserved.

## Known gaps and local setup

- The 32 old in-progress claims remain. Do not take them over without the
  owner's word.
- A zero-health frame without a following death can lose a lent follow.
- Custom settle shapes are forgotten after ten minutes without being asked
  for, even while busy. The existing perception checks outside behaviours
  still use a tolerant absolute three-second difference.
- Nine saved `goal/<hash>` instances warn as unknown on controller start.
  Ask the owner before deleting settings.
- App tests have known timing/race flakes (TKT-01M6ESNXKF and
  TKT-01M6EWJ9). Read retained diagnostics before a focused rerun.
- This worktree's ignored `vm.local` and `plugin/game.local.props` were copied
  from existing worktrees without exposing their contents. Never print local
  configuration, panel tokens, CPA keys or server passwords.
- Do not kill `renki-ctl` on the VM. Write `controller.local.json` as ASCII
  without a BOM. Use the lease for mutations; read-only inspection needs none.
