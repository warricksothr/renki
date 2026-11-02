# Handoff

The state of the work at the last handoff, and what the next agent needs
that the tickets and the roadmap don't already say. Replace this page at
each handoff rather than appending to it.

Handoff of 2026-11-02, from the planning sessions (`agent:claude-code/renki-plan`
and its `-mN` sub-agents).

## Read first

1. AGENTS.md: the rules for `just`, `just ci`, `just review` and tickets.
2. docs/ROADMAP.md: the milestones, their SMART exit goals, and "Working in
   parallel", which says how tickets are handed out and where they conflict.
3. docs/TESTING.md and docs/VM.md before any check or live work.

## What landed

- **#155** filed the 2026-11-01 code review as tickets under TKT-01M6DCF2AB
  (Code review 2026-11-01: robustness, trust and gate gaps).
- **#156** fixed the review's urgent defects: plugin task exceptions are
  contained and logged, Delays stay off the ErrWait clock, the OBS connection
  leak, gather waits and stow safety.
- **#157** wrote docs/ROADMAP.md (M2 to M7 with SMART goals) and groomed every
  milestone into one-PR tickets with files, verification and the `live` label.

The three went up as a stack, and #156 and #157 merged into their stacked
bases after #155 had already landed. Neither reached main. The pull request
that adds this page carries both onto main, with main's pantry follow-up
(#154) merged in. If you are reading this on main, that is done.

**Not deployed.** #156 changes the plugin and the controller, and no session
has deployed it. The last ticket note (TKT-01M6AT4E, Keep the pantry stocked
through idle food production, 2026-11-02 05:52) records controller 75db0ae and
plugin 6e23e2d on the VM. Deploy only under the rules for live work below.

## The queue

`git ticket ready` holds three tickets, all offline, so all three can run at
once. #156 already fixed part of two of them; its notes on each say what it
did. What is left:

| Ticket | Milestone | What is left |
|---|---|---|
| TKT-01M6DCFSRN (OBS adapter: leaked connection per poll, unchecked source restore) | M4 | The source-loss trial restores the window with its own context and fails if the restore fails; `configure` repairs an existing capture source. |
| TKT-01M6DCFSEG (Behaviours: stow walks avoid hostiles; marker health baseline) | M3 | Marker food and cargo detect an injury after healing past their start health, with heal-then-hit tests. |
| TKT-01M638V1XC (Follow: stuck outside a wall, the leader inside, path straight through) | M3 | All of it. Read its 14 notes first, because the movement work it waited on landed in PR 117. |

Everything else is in `draft`, about 130 tickets. The roadmap's wave 1 lists the
ones to promote first, starting with TKT-01M6EF5DTG (VM: exclusive lease for
deploys and live trials). Only the owner promotes a draft. Name the ones you
think are startable, say why, and let the owner choose.

## Waiting on the owner

The roadmap's "Decisions for the owner" lists ten. The first two block
everything else:

1. **Promote wave 1**, the VM lease first.
2. **Release the stale claims.** 32 `in-progress` tickets and 2 in `review`
   carry claims from the 10-29 to 10-31 sessions (`agent:codex/renki-milestones`,
   `agent:claude-code/renki-groom`, `agent:codex/renki-resume`). No session is
   working them. Don't take one over without the owner's word. TKT-01M628X134
   and TKT-01M628X14M are done and only need closing.

The goals marked (proposed) in the roadmap are proposals until the owner
confirms them. Don't treat them as settled.

## Dates

The milestone targets in the roadmap, and the epics' due dates, are guideline
targets: the time-bound part of each SMART goal, not estimates. Don't add
effort estimates or dated deadlines to tickets. A review happens when a wave
drains or a target passes. A missed target gets a new target, never a looser goal.

## Rules that are easy to miss

- **Stacked pull requests.** GitHub retargets a child PR to main only if
  merging its base deletes the base branch. Here it didn't, which is how #156
  and #157 missed main. After a base merges, retarget the next PR to main
  (`gh pr edit N --base main`) before merging it, or merge the stack top down
  into one branch and land that.
- **Live work** (label `live`) needs the one Windows VM. It runs one ticket at a
  time and, once TKT-01M6EF5DTG lands, only under its lease. Until then the
  owner decides who uses the VM. Reading from the VM (audit windows, report
  recipes, read-only panel requests) needs no lease.
- **Secrets.** Never print or ask for the panel token, the CPA key, the server
  password, or the contents of any `*.local.*` file.
- **On the VM**, don't kill `renki-ctl` (port 18765). Write
  `controller.local.json` as ASCII with no BOM.
- **Tickets.** Name yourself with `--actor agent:tool/session` on every write.
  `.checks/` logs are local to the machine that ran them, so put the result
  in the ticket note, not just the path.

## Points that stand

The #157 review left two points standing, argued in its PR description:

- **No calendar floor for reviews.** A stalled wave has no time-based review;
  only a passing target forces one. This is deliberate, as the owner asked for
  no time-based scheduling.
- **Two fallback notes** on TKT-01M656NE4C (the isolated combat world for goal
  3.1). The second supersedes the first, and notes can only be appended.
