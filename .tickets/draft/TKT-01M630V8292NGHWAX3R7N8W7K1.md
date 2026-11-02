---
schema: 4
id: TKT-01M630V8292NGHWAX3R7N8W7K1
title: "Voice: never promise an action no tool can carry out"
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - llm
  - chat
  - controller
assignees: []
milestone: M6
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T05:06:42Z
updated_at: 2026-11-02T15:53:31Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
extensions: {}
---

## Description

Seen 2026-10-28 (23:09): asked to drop something, the voice said it
would drop the wood and proposed `stop`, since drop was not in its tools.
Players believe what the bot says. The prompt should say so, and the
brain can catch the worst cases: a reply that promises an action while
the proposal is empty or a different task.

This is M6's honesty floor: the agent gets more tools in this milestone, and every one of them is worthless if the words around a tool call are not true.

### Where it happens

- `prompt` in `internal/voice/voice.go` tells the model to call a tool "if they ask for something one of your tools does", but never says not to agree otherwise.
- `converse` in `internal/chat/brain.go` says `reply.Say` whatever the proposal was, and `propose` drops a refused proposal with only a log line, so the bot's words stand even when nothing happens.

### Approach

- Prompt: agree to do something only by calling the tool for it in the same reply. If no tool does it, say plainly that the bot cannot do that yet.
- A check in the brain, after the turn: if the reply text promises an action (a small, tested list of first-person future phrases such as "I will", "I'll", "on my way", "heading", mapped to the tasks they imply) and there is no proposal, or the proposal was refused, or it is a different task, replace the words with an honest line ("I can't do that yet." or the refusal reason from `explain`). Log the replacement so the agent report can count it.
- Keep it conservative: a false positive costs a flat reply, a false negative costs trust.

### Files

- `internal/voice/voice.go`: `prompt`
- `internal/chat/brain.go`: `converse`, `propose`, `explain`
- `internal/chat/brain_test.go`

### Verification

- `just check controller test-packages ./internal/chat ./internal/voice`: the 2026-10-28 drop case (reply promises dropping, proposal is `stop`) is replaced; a promise with a matching accepted proposal is kept; a promise whose proposal is refused is replaced with the reason; small talk with no promise is untouched.
- The honesty share in the voice-eval scenarios of TKT-01M630V80T (Spike: compare voice models on fixed chat scenarios), run before and after, recorded in a note.
- `just check ci`.

## Acceptance criteria

- [ ] The prompt tells the model to agree to an action only when it calls the tool for it
- [ ] A reply that says it will do something while proposing nothing, or something else, is caught and replaced
- [ ] Requests it has no tool for get an honest answer, such as that it cannot do that yet
- [ ] Each replacement is logged with the original words and the proposal, so the agent report can count them
