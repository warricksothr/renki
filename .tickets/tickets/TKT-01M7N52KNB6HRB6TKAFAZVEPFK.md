---
schema: 4
id: TKT-01M7N52KNB6HRB6TKAFAZVEPFK
title: "VM: doctor, setup-worktree and up recipes for bringing the bot back"
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - vm
  - tooling
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/refresh-setup
  branch: t3/refresh-bot-setup-automation
  worktree: /home/sothr/.t3/worktrees/renki/t3-527f19de-ed082ea71ff42b98
  commit: a29b23a3e622efa384090c04bd624f4ea479fb6c
  session: null
  claimed_at: 2026-11-17T16:22:39Z
  expires_at: null
archive: null
created_at: 2026-11-17T16:22:39Z
updated_at: 2026-11-17T16:27:49Z
created_by:
  id: agent:claude-code/refresh-setup
  name: ""
updated_by:
  id: agent:claude-code/refresh-setup
  name: ""
extensions: {}
---

## Description

On 2026-11-17 the bot had been offline for two weeks. Finding out why the panel
timed out, which build was installed and whether this worktree could deploy took
a dozen hand-written SSH probes. The cause was Tailscale signed out on the VM,
which left the controller bound to an address that no longer existed. The fresh
worktree also had no `vm.local` or `plugin/game.local.props`.

One read-only recipe should report all of that and name each fix, and a fresh
worktree should get its ignored local files with one command.

## Acceptance criteria

- [x] just vm doctor reports worktree files, SSH, desktop, tasks, game files, Tailscale, panel, controller, build, lease, disk and video, each with a fix, read-only
- [x] doctor exits 1 on a failed check and prints no token, key or address
- [x] just setup-worktree copies missing ignored local files from another worktree without printing contents
- [x] just vm up runs doctor and refuses to start anything when a check failed
- [x] Offline tests for both run in just ci
- [ ] just vm up brings Tas into the world on the VM

## Implementation plan

doctor.ps1 gathers secret-free facts over SSH stdin (nothing copied to the VM); doctor.py judges them in a pure evaluate() covered by doctor-test.py. setup-worktree.sh copies vm.local and plugin/game.local.props from the main checkout or the newest sibling worktree. vm up = doctor then the existing online recipe. Both tests join just ci; docs/VM.md gains a Start here section.

## Notes

**agent:claude-code/refresh-setup** at 2026-11-17T16:27:49Z

Criterion 6 (just vm up brings Tas into the world) is unticked: starting the game waits on the owner's go-ahead. Doctor run live on 2026-11-17 after the owner signed Tailscale back in: 0 failed, warnings for origin/main lacking installed f254e95 (PR #179 open) and for stopped relay/capture tasks. just check ci passed in 302 s.
