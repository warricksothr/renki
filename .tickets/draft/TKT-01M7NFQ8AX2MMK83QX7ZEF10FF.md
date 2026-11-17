---
schema: 4
id: TKT-01M7NFQ8AX2MMK83QX7ZEF10FF
title: Read-only tailnet video feed for a remote OBS compositor
type: task
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - vm
  - safety
  - tooling
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M67CF3AMXYJJB2PYR790ZDH5
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-17T19:28:41Z
updated_at: 2026-11-17T19:28:41Z
created_by:
  id: agent:claude-code/ledger-obs
  name: ""
updated_by:
  id: agent:claude-code/ledger-obs
  name: ""
extensions: {}
---

## Description

The owner streams development sessions from their own PC (TehBeast) with a separate OBS. That OBS composes its own scenes: game, log tail, dev console, and on-demand operator and bot views. Today the only way to show the bot's picture there is the `/view` page as an OBS browser source. That brings the page's whole layout, depends on WebRTC/H.264 playing inside OBS's built-in Chromium, and can't be cropped or used as picture-in-picture.

The better source is the relay's video itself. It is already safe to show: it is the VM OBS's Program output, which the adapter switches to the standby slate whenever capture isn't verified. But everything on the VM is loopback-only by design (RTSP 127.0.0.1:18554, WHEP 127.0.0.1:18889, path `renki`, read password, bounded readers). docs/STREAMING.md says a remote compositor "would need separate transport/auth design". This ticket is that design and its implementation.

### Proposal

- Add one **read-only** listener on the VM's tailnet address for path `renki`: RTSP over TCP, or SRT if that proves simpler through Tailscale.
- Authorize reads by **tailnet source address**, with a MediaMTX `authInternalUsers` entry: `read` on `renki`, `ips` limited to the compositor's tailnet IP. No password then has to live in the compositor's OBS settings or in snapshots of them.
- Add **one reader slot** for the compositor, about 3 Mb/s at the current 1280×720/30/3000 Kbps profile.
- Leave everything else as it is: publishing stays loopback-only, WHEP stays behind panel auth, the panel and `/view` are unchanged, and no new OBS plugin (NDI etc.) goes on the VM.
- Make it opt-in in the relay config, off unless a compositor address is configured, and report it in `just vm doctor` / `windows-status` without printing addresses.

### Consumer side (for context; lives outside this repo)

The owner's OBS adds a Media Source pointing at `rtsp://<blight tailnet name>:<port>/renki` over TCP. It's used in its Bot View layout and as an optional picture-in-picture. Its snapshot tooling already strips URL fragments, and would also strip `user:pass@` if credentials were ever needed.

### Open questions

- RTSP or SRT over Tailscale: latency, and reconnect behaviour when the VM restarts the relay.
- Whether the compositor's address belongs in the private local config (like the panel address) rather than anything checked in.
- How revocation should look: remove the address and reload the relay?

## Acceptance criteria

- [ ] Relay serves path renki read-only on the tailnet address, off unless a compositor address is configured
- [ ] Reads are allowed only from the configured compositor address; publish and WHEP stay loopback/panel-auth only
- [ ] Reader limit has a slot for the compositor without starving panel/view viewers
- [ ] A host OBS Media Source plays the feed over TCP and recovers after a relay restart
- [ ] doctor/windows-status report the listener state without printing addresses or credentials
- [ ] docs/WINDOWS-VIDEO.md and docs/STREAMING.md describe the listener, its auth and how to revoke it
