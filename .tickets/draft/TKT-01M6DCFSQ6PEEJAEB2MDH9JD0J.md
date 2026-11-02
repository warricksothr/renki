---
schema: 4
id: TKT-01M6DCFSQ6PEEJAEB2MDH9JD0J
title: "Panel: refuse public binds, session cookies, safe model baseURL"
type: bug
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - controller
  - safety
assignees: []
milestone: null
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:42:34Z
updated_at: 2026-11-02T05:42:34Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-review
  name: ""
extensions: {}
---

## Description

Panel exposure is bounded only by its token. That is fine on loopback or Tailscale, and weak anywhere else.

### Where

- `internal/config/config.go:273-279` accepts `panel.listen` of `:port` or `0.0.0.0:port`.
- The panel serves plain HTTP, and the cookie value is the raw token, valid for 30 days.
- A failed login sleeps 1 s but limits nothing across concurrent requests.
- `internal/app/panel.go:234`: the `http.Server` sets no `ReadHeaderTimeout`, `ReadTimeout` or `IdleTimeout`. Together with the OBS adapter leak (TKT-01M6DCFSRNNFS3MJHW1P8CDCNB, OBS adapter: leaked connection per poll), that keeps connections open forever.
- `model.baseURL` can be set from the panel to any URL, and the voice sends the API key there (`internal/voice/voice.go:136,218`). Anyone holding the token can therefore redirect the key.

### Fix

- Refuse, or warn loudly, when the panel binds outside loopback or 100.64.0.0/10, unless an explicit `panel.allowPublic` is set.
- Store a random session ID in the cookie, not the token.
- Require `https` for `model.baseURL` unless the host is loopback.
- Set server timeouts.
- Add a small failed-login limiter.

## Acceptance criteria

- [ ] Binding the panel outside loopback/Tailscale needs an explicit opt-in
- [ ] The cookie carries a session ID rather than the token
- [ ] model.baseURL must be https unless loopback
- [ ] The panel http.Server sets header, read and idle timeouts
