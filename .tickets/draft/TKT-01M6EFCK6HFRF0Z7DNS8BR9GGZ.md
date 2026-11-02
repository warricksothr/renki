---
schema: 4
id: TKT-01M6EFCK6HFRF0Z7DNS8BR9GGZ
title: "Voice: query tools for nearby, recall, behaviours and recent events"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - llm
  - terva
  - controller
assignees: []
milestone: M6
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies:
  - TKT-01M62VXZAV8CYT5K4DYAQEZC2Q
  - TKT-01M630V80TFBMWT1B3XY4KP2QJ
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:52:29Z
updated_at: 2026-11-02T15:52:29Z
created_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
extensions: {}
---

## Description

The query half of the tool registry in TKT-01M62VXZAV (Voice: tool registry with groups, query tools and real results). M6's first outcome is that the agent looks before it acts. The registry ticket brings the registry and the two simplest queries (status and inventory). This one adds the queries that need memory and the behaviour engine.

### Tools

All in the Query group, free, read only, offered to owners and non-owners alike (subject to what the registry ticket decides for non-owners):

- **nearby**: a short summary of the latest perception: creatures (with hostility), players, resources and drops within a radius, nearest first, capped at about 10 lines. Say how old the perception is.
- **recall**: places, work sites, chests and their remembered contents, each with its age. Takes an optional item or place name to narrow it.
- **behaviours**: the behaviour instances with template, enabled, running or idle, and last outcome.
- **recent_events**: the last N events the bot saw (joins, leaves, damage, behaviour finished or failed), with times. Until the wake ticket's event bus exists, read the brain's and engine's existing events.

Answers are short summaries, never raw streams, and each is bounded (for example 1,500 characters) so a query cannot blow the token budget.

### Look before acting

Extend the voice-model scenario set from TKT-01M630V80T (Spike: compare voice models on fixed chat scenarios) with at least 5 scenarios whose right answer needs a query first ("is there wood in the chest?", "go to where I chopped", "what is that noise?"). The scenario score counts a turn as looking first when a query call precedes the first action call.

### Files

- `internal/voice/voice.go`: the registry (after TKT-01M62VXZAV)
- `internal/memory/containers.go`, `internal/memory/landmarks.go`, `internal/memory/players.go`, `internal/memory/pickups.go`: what recall reads
- `internal/session/perception.go`: the latest perception for nearby
- `internal/behaviour/engine.go` (`List`): behaviour instances
- `internal/chat/status.go`: how the existing summary is phrased

### Verification

- `just check controller test-packages ./internal/voice ./internal/chat` with fakes for memory, perception and the engine: each query returns real data, respects its size cap and states ages.
- The scenario recipe from TKT-01M630V80T, run against the chosen model, reports the look-first share for the query scenarios.
- `just check ci`.

## Acceptance criteria

- [ ] nearby, recall, behaviours and recent_events return real data from perception, memory and the engine, each with its age and under a size cap
- [ ] Unit tests with fakes cover each query, including empty memory and stale perception
- [ ] The scenario set has at least 5 look-first scenarios, and the replay report shows the share where a query precedes the first action
