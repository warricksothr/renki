# valheim-agent

A machine player for Valheim. It runs as a second Valheim client, on its own
Steam account, on a Windows VM. It joins your server like anyone else and
takes commands and conversation through in-game chat.

It isn't LLM-driven. Movement, gathering and combat are ordinary code. An LLM,
reached through [terva](https://terva.sh), gives the bot a voice in chat and
turns requests it doesn't recognise into tasks from a fixed menu.

## How it fits together

```
Valheim client (bot's Steam account, Windows VM)
  └─ plugin/         C# BepInEx plugin: reads game state, carries out tasks
        │  WebSocket, JSON — docs/BRIDGE-PROTOCOL.md
        ▼
cmd/controller       Go: decisions, task queue, chat routing, safety
        │
        ▼
terva                LLM calls, personas, provider configuration
```

The plugin owns anything that needs frame-by-frame feedback (steering, stuck
recovery, swinging until a tree falls). The controller owns every decision.
The LLM only produces words, plus at most one task from the menu, and the
controller checks that task before it goes anywhere.
[docs/BRIDGE-PROTOCOL.md](docs/BRIDGE-PROTOCOL.md) is the contract between them.

The plugin has to be C# because BepInEx loads .NET assemblies into Valheim's
Mono runtime. Everything outside the game is Go.

## Status

Design only. The protocol is drafted and the Valheim APIs it names were checked
against the game's assembly. Milestone 1 is filed as tickets: the bot joins,
follows, stops, reports status and chats. Run `git ticket list --status draft`
to see them.

## Prior art

The architecture follows
[Dahuchsi/ValheimAI](https://github.com/Dahuchsi/ValheimAI): a plugin inside a
real client, dialling a separate controller over a localhost WebSocket. That
repository has no license, so no code from it is used here.

## License

GPL-3.0-or-later; see [LICENSE](LICENSE). Dependencies keep their own licenses,
all compatible with the GPL: terva and Harmony are MIT, and BepInEx is
LGPL-2.1. Valheim's own assemblies are only referenced at build time and are
never distributed with this project.
