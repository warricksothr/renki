---
schema: 4
id: TKT-01M61T36CCV76K49ZDHSNV04RB
title: "Spike: WebSocket client inside Valheim's Mono runtime"
type: spike
status: blocked
status_reason: Needs Valheim running with BepInEx and the renki plugin, on the dev box or the VM, to watch the bridge connect and reconnect under Mono.
priority: high
due_on: null
labels:
  - plugin
  - protocol
assignees: []
milestone: M1
parent: TKT-01M61T36B839WT7KEZ14TDMB8B
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/groom-start-renki
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 94c0949e9abff4d3c2953c782958e7a0cbd1595c
  session: null
  claimed_at: 2026-10-28T18:37:51Z
  expires_at: null
archive: null
created_at: 2026-10-28T17:49:28Z
updated_at: 2026-10-28T18:50:44Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Everything depends on the plugin holding a WebSocket open from inside the
game. Find out whether `System.Net.WebSockets.ClientWebSocket` works under
Valheim's Mono, or whether we need websocket-sharp or another library.
docs/BRIDGE-PROTOCOL.md section 12, question 1.

A throwaway plugin is fine. It doesn't need the VM; any Valheim install with
BepInEx will do.

## Acceptance criteria

- [ ] A test plugin in a running game exchanges messages both ways with a test server
- [ ] It reconnects after the server restarts
- [x] The choice of library and the reason are recorded in a note

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T18:04:08Z

Grooming, 2026-10-28. The dev box Valheim install (/mnt/e/SteamLibrary/steamapps/common/Valheim) has no BepInEx folder, only a ValheimPlus directory, so the spike needs BepInExPack_Valheim installed there first (or into an r2modman profile), or it waits on the VM. It does not need the plugin scaffold: a throwaway project is fine, and its findings should feed the scaffold ticket.

**agent:claude-code/renki-groom** at 2026-10-28T18:50:44Z

Choice: System.Net.WebSockets.ClientWebSocket. It is in the System.dll the game ships under valheim_Data/Managed (checked on 2026-10-28), so there is nothing to bundle, and it is what Renki.Core.Bridge.BridgeClient uses. The same class passes interop tests on .NET 8 against the Go controller: handshake, chat to tasks, pings, a controller kill and restart with reconnect, bad token. Still unchecked is Unity's Mono specifically. If it misbehaves in game (receive hangs or fragmented frames are the known Mono failure modes), BridgeClient is the only class to replace, and websocket-sharp (MIT) is the fallback.

**agent:claude-code/renki-groom** at 2026-10-28T18:50:44Z

in-progress to blocked: Needs Valheim running with BepInEx and the renki plugin, on the dev box or the VM, to watch the bridge connect and reconnect under Mono.
