# Propria runner and federation receipt — 2026-10-15

Owner scope: consolidate personal Propria tools; keep third-party identities and
existing selectable skills, commands, thinking methods and local hooks. The
in-conversation method picker is deferred, recorded as
`todo:propria_in_chat_method_picker_20261015` in Docstore.

## Live runner

URL: https://mcp.mitechconsult.com/runner/

Coolify application: `bd12dh0rta210otkr98xlxuv`. Runtime source commit:
`9cc35dfa7` on `codex/propria-tool-runner-20261015`. ContextForge remains 1.0.4.
Browser users authenticate through the existing Authentik provider; machine
clients retain their ContextForge tokens. Secrets stay in managed environment
configuration, never in browser JavaScript or this repository.

The owner confirmed successful sign-in and tool-list loading. Real gateway tool
calls passed separately. A signed-in browser Run-button execution has not been
independently observed; list loading is not proof of that final interaction.
Sessions are held in memory by one worker: redeployment requires signing in again.

## Federation readback

The Propria virtual server `320572d2bdd440f3ac61c8b64418d7b3` has **105 tools**.
This pass removed 64 vendor tool associations from the preceding 167-tool
bundle after read-only calls passed on their separate endpoints. No upstream
tool was deleted. Authentication, resource membership and prompt membership were
preserved. An initial overly strict verification compared ordered prompt arrays
and derived fields; set comparison confirmed membership was unchanged.

After vendor separation left 103 tools, two personal LLM Probes tools were
registered using ContextForge's existing REST integration, with descriptions
generated from `llm_probe_tools.py` docstrings:

- `llm-probes-tracked-models`: GET the existing tracked-model catalog; real MCP
  call returned 82 stored entries.
- `llm-probes-results-summary`: GET aggregate stored probe outcomes; real MCP
  call returned counts covering 255 models. These historical outcomes are not
  a new live-model test.

Both use fixed upstream paths, no caller-controlled URL, empty input schemas,
read-only annotations, and disabled REST passthrough exposure. Provider keys
remain in the existing service. No probe execution or provider mutation tool
was added. Direct GET `/providers` failed and was deliberately not registered;
the two admitted routes passed both direct reads and actual gateway calls.

| Separate service | Virtual server | Tools |
|---|---|---:|
| Context7 | `afab426b3755478080a349052f99254a` | 2 |
| n8n documentation | `4cd2eda15e634147a07f9c9848d97e6e` | 3 |
| Cloudflare documentation | `cdb7aa4198bc4fc394ee8b11037559d6` | 2 |
| Glide Data Grid documentation | `4c217785d57f48da9b12014eec7d91e7` | 3 |
| n8n | `c961807d29e24cd790987ff05d940e7c` | 54 |
| Family Court Console (existing separate surface) | `0b85f64905be468ba477d75cb2a80323` | 31 |

Each endpoint is `https://mcp.mitechconsult.com/servers/<virtual-server>/mcp`.
OAuth resource discovery and native-token tools/list passed for all rows and
Propria. Real read-only calls passed for Context7 library lookup, n8n workflow
search, n8n documentation search and Cloudflare documentation search in this
pass; Glide documentation search passed in the preceding pass.

**Owner correction: Agno is retired.** The newly created Agno docs server
`e0eac5bfa47a408da319074f8edfebfb` and gateway
`7f1ff2754f7d49d2bc6f84fad1cd4def` were disabled. Agno associations were removed
from active bundles including dev-docs. Configuration history was retained.
Do not run the exploratory separation script unchanged: its earlier plan
included Agno and is superseded by this owner correction.

## Verification and limitations

- Propria calls passed: docstore-health, memsearch-status,
  octopoda-octopoda-status, coolify-write-list-projects, a synthetic read-only
  mem-query returning 42, and propria-runner-open-runner. Successful calls do
  not establish the health of every underlying subsystem or every write tool.
- The atomic-tools catalog returned 55 operations across nine families; this
  verifies discovery, not execution of those operations.
- Twelve backend tests and detached UI tests passed. UI tests cover search,
  schema input fields, value types, omission, conversion preservation, refresh,
  single submission and result display.
- The earlier Advocatio case-claim-gaps output-schema/structured-output failure
  remains unresolved. No destructive or case-data write tool was exercised.
- Remote endpoints are not newly installed ChatGPT plugins. Existing desktop
  plugin installations and their selection menus are preserved.
- Search/CCC indexes, filesystem access and session capture require their local
  runtime. Skill-only plugins remain installed; hosting MCP tools is not a
  replacement for their selectable instructions.
- Provider-managed cloud connectors cannot be transferred by copying their
  account authentication. CourtListener and the external n8n-mcp service need
  provider authorization; prior Engram and Cloudflare management endpoint probes
  encountered provider-side HTTP 403. Cloudflare docs are a separate service.

Local nonsecret operational readbacks are under
`.runtime/gateway-auth-20261015/`: `vendor-separation-receipt.json`,
`current-endpoints.json`, and `propria-call-verification.json`. They are
machine-local receipts, not source files or credentials.
`llm-probes-verification.json` records the subsequent two admitted tools;
the vendor-separation receipt's 103 count is the intermediate state.

Byline: Codex · GPT-6 · 2026-10-15. This receipt records observed state and
owner decisions, not completion of every personal-plugin migration.
