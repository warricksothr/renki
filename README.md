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

- **M1** (join, follow, stop, status, chat) was proven in the game on 2026-10-28.
- **The panel** is in daily use.
- **M2** (hauling and gathering wood) and **M3** (surviving danger and standing with the owner) are in progress. Many of their parts work in the game, but none of their exit goals is met yet.

[docs/ROADMAP.md](docs/ROADMAP.md) has the milestones, their dates and the measurable goals that close each one. `just ci` covers everything that runs without Valheim, and docs/TESTING.md describes the checks in the game.

## Working on it

Every task is a `just` recipe; `just` lists them. `just ci` is the gate,
`just controller fake` talks to a controller with no game running, and
`just package` builds the bundle for the Windows VM. AGENTS.md has the
conventions and the ticket workflow.

`just ci` needs what CI (.github/workflows/ci.yml) installs: Go at the
version in `go.mod`, the .NET 8 SDK, Node.js 20+, Python 3, `just`,
[`git-ticket`](https://github.com/terva-sh/git-ticket) for the ticket check,
and Playwright's Chromium for the panel UI suite (`just panel ui-install`, plus
`just panel ui-install-system` for its OS libraries). The Renki.Plugin DLL
also needs a Valheim install (plugin/Directory.Build.props); without one,
`just ci` says it skipped it.

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
