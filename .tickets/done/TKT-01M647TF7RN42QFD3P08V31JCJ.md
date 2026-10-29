---
schema: 4
id: TKT-01M647TF7RN42QFD3P08V31JCJ
title: Keep development checks concise and retain full logs locally
type: chore
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - tooling
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T16:27:51Z
updated_at: 2026-10-29T16:45:29Z
created_by:
  id: agent:codex/renki-resume
  name: ""
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

Reduce conversation context spent on routine development: provide focused component checks and a recipe that captures complete check output locally, emits concise success summaries, and surfaces bounded failure diagnostics without masking exit codes. Document when to use focused checks and when just ci remains required.

## Acceptance criteria

- [x] Focused checks avoid rebuilding unrelated components and unnecessary interop setup
- [x] A concise wrapper retains complete logs and preserves failures
- [x] The workflow is documented and just ci passes

## Implementation plan

Add a root check wrapper retaining unique local logs with bounded failure tails and original statuses, focused controller-package and Core-only plugin recipes, and a documented workflow. Verify success and failure logging, then run the full gate through the wrapper.

## Notes

**agent:codex/renki-resume** at 2026-10-29T16:33:33Z

Added just check to keep unique complete .checks logs and print a result or an eighty-line failure tail. The capture uses a pipe so children cannot truncate the log by opening /dev/stderr. Added just controller test-packages and just plugin test-core, plus wrapper regressions in just check-test (part of just ci). Documented focused iteration, task-filtered audits, saved diagnostics and ticket handoffs in AGENTS.md and docs/TESTING.md. just check ci passed before the final capture hardening; just check check-test passes with the hardening, including failure status 37 and both log streams. A final full gate follows this ticket update.

**agent:codex/renki-resume** at 2026-10-29T16:45:29Z

Added just vm panel for repeatable live API checks over SSH, keeping the login token on the VM. It syncs panel-request.ps1 and accepts GET/POST /api paths plus an optional JSON body. Read-only /api/status was verified against the current VM. docs/VM.md documents it.

## Summary

Completed concise check capture, focused component recipes, logging regressions and workflow documentation. Final just check ci passed with zero build warnings/errors and all 122 C# tests; full CI output was retained as 55 log lines while the conversation received two lines. Success/failure capture, status propagation, unique logs and protection from /dev/stderr truncation are verified by just check-test.
