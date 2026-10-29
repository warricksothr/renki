---
schema: 4
id: TKT-01M630V7ZTEW2WR52HVWXR475M
title: "Panel: choose the voice's model at runtime, to compare them"
type: task
status: ready
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - llm
  - terva
assignees: []
milestone: null
parent: TKT-01M628X106RD9FFR8YBXV7K5HT
origin: null
dependencies:
  - TKT-01M62VXYR69PKZGJFEWQ1759GV
  - TKT-01M628X13X2G0849113H4BSRP3
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T05:06:42Z
updated_at: 2026-10-29T05:26:35Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The voice's model is fixed in the controller's config (`llm.terva`). On
2026-10-29 cpa-api gained an Anthropic key, so `haiku-5-5` is available
beside the local `gemma-4-12B-heretic-v3-mtp` the VM uses now, and a
stronger local model may join them. For testing, the panel should switch
between them while the bot runs.

- The settings store holds the chosen provider, model and base URL; the
  config file seeds it. Keys stay in environment variables.
- The panel lists what each backend offers (cpa-api's `/v1/models`) and
  switches the voice; the next reply uses it.
- Each reply in the audit log records the model, the latency and the
  tokens, so they can be compared from the log.

## Acceptance criteria

- [ ] The panel lists the models the configured backends offer and switches the voice to one without a restart
- [ ] Every reply in the audit log names the model that wrote it, with its latency and tokens
- [ ] The choice is kept in the settings store, recorded with who made it
