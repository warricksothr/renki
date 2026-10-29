# renki

*Renki* is Finnish for a farmhand: someone who does the chores and follows the
boss around. This one is a machine player for Valheim. It runs as a second Valheim client, on its own
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

Milestone 1 is written and waiting to be tried in the game. The bot can
join, follow, stop, report its status and chat. `just ci` covers everything
that runs without Valheim, including the C# bridge client talking to the real
Go controller. [docs/TESTING.md](docs/TESTING.md) is the in-game checklist,
and `git ticket list --status review` lists the tickets waiting on it.

## Working on it

Every task is a `just` recipe; `just` lists them. `just ci` is the gate,
`just controller fake` talks to a controller with no game running, and
`just package` builds the bundle for the Windows VM. AGENTS.md has the
conventions and the ticket workflow.

Development checks use Go, the .NET SDK, `just`, and Node.js 20+ for the
panel's terrain maths (`just controller test-map`).

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
