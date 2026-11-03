# Handoff

The state of the work at the last handoff, and what the next agent needs
that the tickets and the roadmap do not already say. Replace this page at
each handoff rather than appending to it.

Handoff of 2026-11-03, from `agent:codex/portal-follow`. The owner selected
TKT-01M6FT1ZVW, **Follow: go through the portal the leader just used**, from
the previous handoff's drafts.

## Read first

1. AGENTS.md, for recipes, tickets and committed review.
2. docs/ROADMAP.md, especially working in parallel and the conflict hotspots.
3. docs/PORTALS.md, for portal following and the remaining general travel work.
4. docs/BEHAVIOURS.md, docs/TESTING.md and docs/VM.md before live work.

## Current change

Branch `t3/continue-renki-bot` implements portal following. The ticket carries
its plan, checks and remaining live evidence. Its code is ready for review;
it has **not been deployed** in this session.

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

## Deployed state and next live session

Main at the start was 0411381 (#175, the previous handoff). The last deployed
build remains b5cc019 (#174, perception skew), with passive Loot enabled.
Read-only VM status in this session found the controller running, Valheim
stopped, the bot disconnected, and the lease free. No lease was taken and
nothing was started, stopped or deployed.

For portal following, after review/merge, with Nimah available:

1. Take `just vm lease agent:tool/session`, then `just check vm deploy`.
2. Ask Tas to follow. Walk through a connected portal, wait on the far side
   for Tas to arrive and resume following, then do the same trip back.
3. Check a departure while Tas is picking up felling drops on a passive loan.
   The returned follow must still take the source portal.
4. With cargo that native portal rules refuse, confirm Tas explains the cargo
   refusal and stays rather than walking home. Stop/order takeover during an
   approach must win, with no leftover trigger allowance.
5. Retain controller audit/log evidence for `portal departure`, `portal arrived`
   and `portal failed`, plus follow task outcomes. Tick the live criterion
   only after the pair crossing and return pass. Release the lease afterward.

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
or any owner order. Loot and future gathering are passive and on by default.

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
