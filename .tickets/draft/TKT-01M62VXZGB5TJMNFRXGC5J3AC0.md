---
schema: 4
id: TKT-01M62VXZGB5TJMNFRXGC5J3AC0
title: "Voice: a lasting terva session per bot, and Valheim lore"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - terva
  - llm
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
created_at: 2026-10-29T03:40:49Z
updated_at: 2026-11-02T15:55:40Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
extensions: {}
---

## Description

Each voice turn starts a fresh terva runtime with a summary. Give each bot
a lasting session, so it remembers what it asked or promised an hour ago,
pruned so it stays within budget. Put lasting Valheim knowledge (fuel
types, what each machine makes, which tools break what) into the bot's
tervaHome as lore or skills rather than into every prompt.

This matters to M6 twice: a woken agent that forgets its own question an hour later cannot follow up, and M6 bounds tokens per hour, so whatever the session keeps must be pruned to a fixed size rather than grow with the day.

### What exists

- `voice.Reply` (`internal/voice/voice.go`) calls `sdk.New` per turn and `rt.Close()` after it; history reaches the model only as `Request.Recent` chat lines in the prompt.
- terva's SDK runtime (terva.sh/terva v0.141.5, `packages/agent/sdk/sdk.go`) has `Messages()`, `SetMessages()` and `Compact(ctx, instructions)`, so a session can be saved and restored between turns, or one runtime kept per bot.
- `Config.TervaHome` sets `TERVA_HOME`, from which terva adds AGENTS.md, skills and lore to every prompt. The VM uses `terva-home` (docs/VM.md). The voice runs in an empty temp directory so it picks up no repository files.

### What to build

- One session per bot (keyed by world and character), kept in memory and saved to `dataDir` after each turn, restored on start.
- A size limit in tokens (default, for example, 6,000) enforced before each turn by compacting or dropping the oldest turns; the limit is a setting.
- Chat turns and woken turns share the session, so a question asked on a wake is visible when the owner answers.
- A lore directory under the bot's tervaHome with a small, reviewed set of Valheim facts, and a check that the prompt the voice sends no longer carries them inline.
- The per-turn token cost before and after, from the voice records, noted on the ticket.

### Files

- `internal/voice/voice.go`: `Reply`, `New`, `runtime`, `Config.TervaHome`
- `internal/voice/voice_test.go`: the fake runtime, extended with `Messages` and `SetMessages`
- terva SDK `packages/agent/sdk/sdk.go` in the module cache (`go doc terva.sh/terva/packages/agent/sdk.Runtime` via `just controller dependency-doc`)
- `docs/VM.md`: tervaHome on the VM

### Verification

- `just check controller test-packages ./internal/voice`: two turns for one bot share history; turns for two bots do not; the session survives a restart through dataDir; a long session is cut to the limit before the next turn.
- The voice-eval recipe from TKT-01M630V80T (Spike: compare voice models on fixed chat scenarios) run before and after, with mean tokens per turn noted.
- `just check ci`.

## Acceptance criteria

- [ ] Turns for one bot share a session that survives between turns and is kept within a size limit
- [ ] Valheim knowledge lives in the bot's tervaHome and reaches the model without being in the prompt
- [ ] The session survives a controller restart, and sessions of different bots never mix
- [ ] Mean tokens per turn before and after, from the voice-eval recipe, are noted on the ticket
