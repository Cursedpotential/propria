# ChatGPT-Condense project description.md
> _Byline: Claude Code · Sonnet · 2026-07-11_

## Snapshot
- Source tool / export style: standard ChatGPT web export via the **ChatGPT Exporter**
  browser extension (footer credit: "Powered by ChatGPTExporter.com").
- Size: 36 KB · 456 lines · single-turn conversation (one user prompt, one assistant
  response — no back-and-forth).
- Format: Markdown, `.md`, with a structured metadata header block.
- Era/date: **Created 8/3/2025 0:33 · Updated 8/3/2025 3:18 · Exported 8/21/2025 13:28**
  (this is the only one of the three files with real, explicit timestamps).
- Turns: 1 user prompt → 1 assistant reply. The single user "turn" is unusually long
  because it embeds an entire pasted system/tool-schema prompt as context (see below).

## Structure & format
- Header block: `**User:** Matthew Salem (Error 404 Fuks Not Found) (matt.salemnet@gmail.com)`,
  `**Created:**`, `**Updated:**`, `**Exported:**`, `**Link:**` (a `chatgpt.com/c/<uuid>`
  permalink) — all as bold-label Markdown key/value lines.
- Turn delimiters: `## Prompt:` for the user turn, `## Response:` for the assistant turn.
  Only one of each in this file (single-turn chat).
- Embedded artifact: the user's "prompt" is actually two things concatenated —
  (1) a short natural-language ask ("help me condense a long system prompt under 8000
  chars"), followed by (2) the **full ~350-line tool-calling system prompt** the user
  wants condensed, verbatim, including a "SuperAssistant" `<function_calls>` XML schema
  spec and a complete tool catalogue (filesystem.*, notion.*, supermemory.* — MCP-style
  tool definitions with parameter schemas). The assistant's reply is a structured
  Markdown response with a table and three numbered strategy options plus example XML.
- PARSING NOTES:
  - The header block is clean, parseable key/value metadata — best of the three files
    for automated metadata extraction (name, email, created/updated/exported timestamps,
    canonical source URL all present).
  - The pasted tool-schema block inside the "user" turn is **not user-authored
    prose** — a parser must distinguish "user's actual ask" (2 sentences) from "context
    payload the user pasted" (the ~350-line tool spec) or facet-tagging will
    misattribute a code/tool-schema chunk to the "identity/mood" facets.
  - The user's real email surfaces here as `matt.salemnet@gmail.com` (vs.
    `matt.salem85@gmail.com` used elsewhere) — worth noting as an alternate identity/PII
    variant for entity resolution across the corpus.
  - The assistant response includes one small embedded XML code block (an example
    `<function_calls>` invocation) — trivial for a parser to fence-detect.

## Section-by-section breakdown (in order)
1. **User's actual request** (2 sentences): the user has a tool that requires a system
   prompt as input, the prompt is too long, and they want either a compressed version
   under 8,000 characters or an alternative way to reference it.
2. **Pasted context payload**: the full "SuperAssistant" tool-calling system prompt the
   user is trying to shrink — a `<function_calls>`/`<invoke>` XML schema spec (near-
   identical in spirit to the Claude tool-call convention) plus a full tool catalogue:
   `filesystem.*` (read/write/edit/list/move/search files), a large Notion API surface
   (`notion.notion_*` — blocks, pages, databases, comments, users), and `supermemory.*`
   (addToSupermemory / search / fetch — a third-party memory-MCP product). This is a
   snapshot of an MCP/tool-integration stack the owner was assembling in August 2025,
   independent of the custody case.
3. **Assistant's response** (o3 pro, "Reasoned for 2m 15s"): three condensation
   strategies presented as options —
   1. "Aggressive but Safe Condensation" (a markdown table of strip/merge/compress
      tactics targeting 60-70% size reduction),
   2. "Split-and-Reference" (core ≤8k prompt + auxiliary prompt referenced by name),
   3. "External Source of Truth" (store the full prompt as a file the model loads via a
      `filesystem.read_multiple_files` tool call at runtime, shown as a worked XML
      example) —
   followed by "Additional Tips" (versioning/checksums, an automated minifier script
   idea, keeping full text in version control with CI-generated condensed forms,
   fail-loud truncation checks) and a closing offer to do the actual condensation work
   if the user pastes the source text.

## Facets present (extraction-lane map)
| facet | present? | examples / notes |
|---|---|---|
| identity (who) | yes (minor) | User identity: "Matthew Salem (Error 404 Fuks Not Found)," email `matt.salemnet@gmail.com` — a ChatGPT display-name/handle variant, useful for cross-file identity resolution |
| entities | yes | Third-party tools/products referenced: Notion API, Supermemory (memory-MCP product), a generic "filesystem" MCP surface — infra/tooling entities, not case entities |
| relationships | no | — |
| timeline/events | no | Only the file's own creation/update/export timestamps (metadata, not narrative events) |
| life-history | no | — |
| legal-strategy | no | — |
| legal-artifacts | no | — |
| mood/sentiment (owner-self, low-pri) | no | Purely task-focused, neutral tone |
| psychiatric | no | — |
| code/app-dev/plans | **yes (dominant facet)** | Entire file is a tool-integration/prompt-engineering exchange: an MCP-style function-calling schema, a full tool catalogue, and prompt-compression strategy/tooling advice (versioning, minifier scripts, CI-generated condensed prompts) |
| work | no | Not employment-related; this is personal tooling/hobby-project work |

## Notable content
- **No custody/legal-case content whatsoever** — this file is 100% orthogonal to the
  other two in this batch; it documents the owner's personal LLM-tooling stack circa
  August 2025 (filesystem MCP + Notion MCP + Supermemory MCP), which is thematically
  close to this platform's own MCP/tool-registry work but not case material.
- Confirms the owner was already thinking about "condense for context-window limits" and
  "external source of truth via file reference" as of Aug 2025 — directly relevant
  precedent for the current chat-ingest/extraction-lane design discussion.
- Alternate email `matt.salemnet@gmail.com` (vs. the now-current `matt.salem85@gmail.com`)
  — an identity-resolution data point if the corpus spans multiple ChatGPT accounts.

## Sensitivity
- **None flagged.** No PII beyond the owner's own name/email/handle, no abuse,
  psychiatric, or case content. Lowest-sensitivity file in this batch — safe to treat as
  general reference/tooling material.
