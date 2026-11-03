---
schema: 4
id: TKT-01M6FCYSS5K7X8KQ967KMJX500
title: "Passive mode: resume a lent follow after Tas dies mid-run"
type: task
status: archived
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive:
  archived_at: 2026-11-03T00:32:21Z
  from_status: draft
  reason: "Done in PR #173: a death during a passive run hands the lent follow to the brain's resume (Options.ResumeFollow)."
created_at: 2026-11-03T00:29:14Z
updated_at: 2026-11-03T00:32:21Z
created_by:
  id: agent:claude-code/renki-passive
  name: ""
updated_by:
  id: agent:claude-code/renki-lead
  name: ""
extensions: {}
---

## Description

Follow-up to TKT-01M6EWN4WH (Behaviours: passive mode lends an owner's follow to passive instances).

If Tas dies during a passive run, the death revokes the loan, and the engine logs `follow not returned`. The follow is not resumed after respawn and nobody is told. If Tas dies during a plain follow, the chat brain's `leaveResume` restarts it and says "I'm back. Coming to find you". So a death near drops now ends the follow silently, where before it was resumed.

The implementation brief said not to restart the follow after a death. Decide whether a death during a loan should hand the follow to `Home.ResumeAfterDeath`, as the brain does, or at least tell the followed player through Notify.

### Files
- internal/behaviour/engine.go (`giveBack`, the revoked-handle path)
- internal/chat/brain.go (`leaveResume`)

## Notes

**agent:claude-code/renki-lead** at 2026-11-03T00:32:21Z

archived from draft: Done in PR #173: a death during a passive run hands the lent follow to the brain's resume (Options.ResumeFollow).
