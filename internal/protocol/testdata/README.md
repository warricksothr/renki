# Protocol fixtures

Shared by the Go tests in this package and the plugin's C# tests. Each file
is one frame, named for its sender's directory. docs/BRIDGE-PROTOCOL.md is
the authority; when it changes, change these and both test suites with it.

- `plugin/`, `controller/`: valid frames. Decoding one and encoding it again
  must give back the same JSON, compared as values rather than bytes. The
  doc's own examples are copied verbatim (`hello`, `welcome`, `state`,
  `perception`, `inventory`, `chat`, `event_damaged`, `task_failed`,
  `task_follow`); the rest fill in what the doc only sketches.
- `lenient/<sender>/NAME.json`: a valid frame plus fields this version does
  not know. It must decode, and encoding it must give `<sender>/NAME.json`,
  because unknown fields are ignored (section 3).
- `reject/<sender>/`: frames that must not decode.
  - `version_*`: an unknown major version, or none. Reject it as a version error.
  - `unknown_type`: answer with an `error` frame, code `bad_message`.
  - `wrong_direction`: a real type that this sender never sends. Treat it
    like an unknown type.
