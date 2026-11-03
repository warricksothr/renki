# Handoff

The state of the work at the last handoff, and what the next agent needs
that the tickets and the roadmap don't already say. Replace this page at
each handoff rather than appending to it.

Handoff of 2026-11-03, from `agent:claude-code/passive-deploy`. That session
worked with the owner (Nimah in game) on the VM lease, loot reach and passive
mode.

## Read first

1. AGENTS.md: the rules for `just`, `just ci`, `just review` and tickets.
2. docs/ROADMAP.md: the milestones and "Working in parallel".
3. docs/BEHAVIOURS.md: idle and passive modes, the loan, Loot's settle area
   and reach. This is where the current work is.
4. docs/TESTING.md and docs/VM.md before any check or live work, including
   "The VM lease".

## Where things stand

Main is at b5cc019, and **that build is deployed**, with Tas (Tasleroch) in
the world. No pull requests are open.

What this stretch landed, in order:

| PR | What it did |
|---|---|
| #168, #169 | The loot anchor and the piles view; wave 1 work. |
| #170 | VM lease. `stop` and the panel's stop are exempt so a person can always stop the bot; inside `lease.py run` they still guard. |
| #171 | Loot's adaptive reach (TKT-01M6EYEMRV): a settle tracker per player; the reach grows from 8 m to 30 m while the anchor stays in a small area. |
| #172 | Tickets only: the idle/passive decision on epic TKT-01M62VXYQB, plus the drafts listed under "Next". |
| #173 | Passive mode (TKT-01M6EWN4WH). Instances are `idle` or `passive`; Loot is passive and on. While a followed owner is settled, the engine lends the follow to a passive run and restarts the same follow after. A death during the run resumes the follow after the respawn. |
| #174 | Perception skew. A scan sent just after the latest state frame is stamped ahead of it, and behaviours and goals refused it. That made passive Loot get cut off ("condition changed") several times a second. A scan may now lead state by up to 1 s (`behaviour.PerceptionBeside`). |

## The live check in progress

TKT-01M6EWN4WH, "Behaviours: passive mode lends an owner's follow to passive
instances", is done offline. Its acceptance criterion 6, the live check, is
still unticked. Its newest note has what the owner saw on a892cfa:
- several passive runs worked;
- the churn that #174 fixes.

Rerun it on b5cc019 with the owner, under the lease:

1. Owner says "follow", stops near a tree, fells it. After about 5 s settled,
   Tas picks up the drops and falls back in when the owner walks off.
2. The owner drops a stack at their feet. Tas leaves it alone (the anchor
   rule from #169).
3. The owner walks away mid-pickup. The run ends ("followed player moved
   on") and the follow restarts.
4. Any owner order mid-pickup wins, and no follow is restarted.

Read the result from the controller log. Filter `C:/renki/logs/controller.log`
for `behaviour started|follow returned|interrupted|finished|pickup`; each
loan logs `passive=true follow=<id>`. Then tick criterion 6, or note what
failed.

TKT-01M6EF5DTG (VM: exclusive lease for deploys and live trials) is in
`review` for the same reason. Criterion 1 needs its live A/B refusal check;
the ticket's notes give the steps, with `/api/act/home` as the refused
POST.

## Next

All of these are drafts. Only the owner promotes one, so name them and let the
owner choose.

- **TKT-01M6FT1ZVW (Follow: go through the portal the leader just used).**
  High priority, filed in this session after the owner hit it live. The owner
  asked for it ("can we look into following through portals?"), but has not yet
  said to build it. The ticket has the approach. It needs a scoped exception to
  the portal guard, which is owned by TKT-01M6AYP8 (Cache building collisions
  and remember portal links). That ticket is `in-progress` and claimed by
  `agent:codex/renki-milestones`, so expect conflicts in WorldNavigation.cs and
  Driver.cs.
- **TKT-01M6F877TX (Behaviours: passive gather works alongside the
  anchor).** High priority. The owner's stated goal: when they harvest, Tas
  harvests too. It depends on TKT-01M6F877Q8 (Behaviours: infer the anchor's
  activity from changes around them). It also needs a fix for the known
  re-lend churn (below), because it will be the second passive instance.
- **TKT-01M6EYEMTX (Panel: show loot's anchor, settle area, reach and drop
  decisions).** There is no panel card for passive runs yet. The data is
  already in `/api/behaviours`: `PassiveRun{Follow, Player}`.
- **M5, later:**
  - TKT-01M6F877XV (Behaviours: needs ledger ranks passive work by what the
    base lacks);
  - TKT-01M6F8780V (Agent: supply the passive list, with validation and
    built-in fallback). The owner's design: the agent side (`internal/voice`)
    supplies bundles of passive routines, ranked by what the base needs, and
    the engine falls back to its built-in list.

The owner's design for passive mode is recorded on epic TKT-01M62VXYQB. In
short:
- *idle* work runs only when Tas has nothing to do;
- *passive* work runs alongside following, only while the owner is settled;
- passive work yields when the owner leaves the area or gives any order;
- gathering and looting are passive and on by default.

## Known gaps that stand

- **Zero-health frame.** If a frame reports zero HP and no death follows, a
  follow lent at that moment is not restarted and the owner is not told. The
  game is not known to send such a frame. See docs/BEHAVIOURS.md.
- **Re-lend churn.** A higher-priority passive instance that preempts a passive
  run restarts the follow, which is then lent again straight away. This is
  harmless with Loot as the only passive instance. Fix it with the
  gather-alongside ticket.
- **Custom settle shapes.** A custom settle shape is forgotten after 10 minutes
  without being asked for, including while its instance is busy. Its reach then
  starts over.
- **Strict perception checks elsewhere.** marker_food, marker_rest and
  goal/food still use `math.Abs(State.T - Perception.T) <= 3`. This is
  tolerant and not the bug #174 fixed. They could adopt
  `behaviour.PerceptionBeside` for one rule.
- **Unknown goal instances.** At every start the controller warns about nine
  saved `goal/<hash>` instances: "invalid instance … unknown goal". They are
  skipped. Nobody has looked at where they come from. Ask the owner before
  deleting saved settings.
- **Flaky app tests.** `internal/app` tests sometimes time out under load
  (TKT-01M6ESNXKF). Rerun before chasing them.

## Waiting on the owner

- Promoting the drafts above, and whether to build portal following now.
- The live check above, which needs the owner in game.
- The 32 stale `in-progress` claims from the 10-29 to 10-31 sessions are still
  there (earlier handoff, decision 2). Don't take one over without the owner's
  word.

## Rules that are easy to miss

- **The VM lease.** `just vm lease ACTOR [MIN]` before anything mutating, and
  `just vm release` when done. `just vm lease-status` shows the holder.
  Mutating recipes refuse without it. Reading (status, logs, read-only panel
  requests) needs none. A deploy renews the lease.
- **Deploying.** `just check vm deploy` builds and installs the plugin and
  controller, then starts Tas. A worktree needs the gitignored `vm.local` and
  `plugin/game.local.props`. Copy them from another worktree without reading
  or printing them; without the props file the plugin build can't find the
  Valheim assemblies. `just vm stop` then `just vm panel-start` takes Tas
  offline but leaves the panel up.
- **Secrets.** Never print or ask for the panel token, the CPA key, the server
  password, or the contents of any `*.local.*` file.
- **On the VM**, don't kill `renki-ctl` (port 18765). Write
  `controller.local.json` as ASCII with no BOM.
- **Tickets.** Name yourself with `--actor agent:tool/session` on every write.
  `git ticket create --json` can fail to parse even when the ticket was made.
  List before retrying, or you get a duplicate.
- **Merges.** Check that a merge reached main with
  `git merge-base --is-ancestor <head> origin/main`. Stacked PRs don't
  retarget themselves; see the earlier handoff in git history.
