# Prior-Iterations Mining — AI-Chat Parsers/Schemas/Chunkers

> _Byline: Claude Code · Sonnet 5 · 2026-07-11_

Read-only mining pass across the owner's prior-iteration corpus for ChatGPT/Gemini/Claude/Perplexity
chat-format schemas, chunkers, and parsers (OpenCode explicitly out of scope per owner brief — already
covered separately, see `schemas/chat-formats/opencode.json`). Writes were limited to the four provider
descriptor JSON files (mirrored to the plugin cache) and this report. No git push, no deploys, no code
changes to any parser.

## Where I looked

1. **`D:\casebible\iterations_index.duckdb`** (table `iteration_index`, 2,080 rows / 12 iterations) —
   queried via an ephemeral `uv run --with duckdb python` (no persistent duckdb install existed in the
   Agno-MCP-Platform venv; Windows-style path used per this box's DuckDB convention).
2. **`docs/planning/parser-iterations-inventory.md`** — read in full. This prior sweep is scoped to
   *messaging/social-media* parsers (iMessage/SMS/Facebook/SBV) and **explicitly excludes AI-chat
   parsers** ("AI-chat/LLM-transcript parsers (chatgpt/claude/gemini/perplexity) are explicitly OUT OF
   SCOPE" — its line 9). No AI-chat content was extractable from it; it exists purely as a coverage-map
   reference (its "0. Where I looked" table lists the same donor locations this pass also searched).
3. **`server/tools/parsers/ai_chat/`** — 12 current in-repo atomic tools, all thin `@register` wrappers.
   9 of the 12 delegate to a vendored library at `server/vendored/chatminer/parsers/` (see below); the
   other 3 (`claude_ai_export.py`, `claude_code_jsonl.py`) are hand-written, and `claude_code_jsonl.py`
   was verified against a real live sample this pass.
4. **`dev-resources/` and `_stale/`** — grepped for chat-export parser code and Chrome-extension-style
   exporter artifacts (manifest.json, `chrome.runtime`/`chrome.storage`, TypingMind/SaveGPT/ChatGPT-Exporter
   naming). See "Chrome-extension exporters" section below for the (negative) result.

## Key discovery: the current codebase already has a rich, sourced parser library

`server/vendored/chatminer/parsers/` is a vendored library (not third-party `pip`-installed — the
directory is checked into `server/vendored/`) with **9 chat-format parsers**, each with a module
docstring documenting its source format and, for several, an explicit "Adapted from: `<real named
upstream OSS project>`" citation:

| Parser | Product | Adapted from (named, real, public) | This pass's confidence |
|---|---|---|---|
| `chatgpt_official.py` | ChatGPT official `conversations.json` (mapping tree) | gavi/chatgpt-markdown, pionxzh/chatgpt-exporter | already verified (pre-existing descriptor) |
| `chatgpt_share.py` | ChatGPT "Share" link markdown export | rashidazarang/chatgpt-chat-exporter, pionxzh/chatgpt-exporter | **NEW variant added**, high-confidence mined |
| `gemini_chrome.py` | Google Gemini **Chrome extension** markdown export | (docstring: "from your actual files") | **NEW variant added** — this IS the Chrome-extension exporter the owner flagged |
| `gemini_json.py` | Gemini JSON (speculated Google Takeout origin) | none cited — speculative | **NEW variant added**, flagged unverified/speculative |
| `claude_code.py` | "Claude Code" flat `{role,content,timestamp}` JSONL | none cited | not adopted as primary — doesn't match the real CLI session shape found on this box (see below) |
| `claude_md.py` | Claude copy-paste markdown (3 marker-pattern families) | none cited — defensive multi-pattern guess | folded into existing `claude_export_md` gotchas |
| `perplexity_gdpr.py` | Perplexity **official GDPR data export** `conversations.json` | FraYoshi/perplexity-export-convert | **NEW variant added**, upgraded provider to verified |
| `perplexity_md.py` | Perplexity generic markdown fallback | none cited — delegates to `generic_md.py` | folded into existing `perplexity_md` gotchas |
| `perplexity_plugin.py` | Perplexity **browser userscript** export | greasyfork/Perplexity-AI-Chat-Exporter | **NEW variant added**, upgraded provider to verified |

This library was not found anywhere in the DuckDB-indexed donor dumps or `dev-resources`/`_stale` — it
exists only in the current codebase, so none of these 9 parsers themselves are "prior iterations" in the
sense of the owner's brief (multiple old copies scattered across archives). They are, however, the single
richest source of *documented format knowledge* found during this pass, several with real named-upstream
provenance, which is why they drove most of the descriptor enrichment below.

## Per-provider findings

### ChatGPT — 4 distinct prior-iteration versions found, all one lineage

A standalone `chatgpt_parser.py` ("ChatGPT JSON Parser - Sprint 1") script recurs at **4 distinct byte
sizes** across 9+ donor locations found via the DuckDB index:

- 2865 bytes (earliest/stub, no schema-detection) — `dev-resources/Archives/The_Platform_Archive/MCP_BACKUP/platform_archive/mcp-tool-platform/utilities/scripts/chatgpt_parser.py`, `mcp-tool-platform/utilities/scripts/chatgpt_parser.py`
- 2966 bytes — `dev-resources/Archives/Agno-MCP-Platform-alpha/.claude/worktrees/migration-plan-v8/utilities/scripts/chatgpt_parser.py`, `OTHER_RESOURCES_TO_SORT/utilities/scripts/chatgpt_parser.py`, `TEMP_GITHUB_COMPARE/utilities/scripts/chatgpt_parser.py`, `TheBigOne/01_MCP_Tool_Platform_Repo/utilities/scripts/chatgpt_parser.py`, `extracted-code/parsers/chat-exports/chatgpt_parser.py`
- 4352 bytes (adds full docstrings) — `OTHER_RESOURCES_TO_SORT/_project_dirs_loose/chatgpt_parser.py` (+ `pasted_content/` dup), `The_Platform_Archive/MCP_BACKUP/dev_docs_artifacts/chatgpt_parser.py`, `dev_docs_artifacts/chatgpt_parser.py`
- **23603 bytes — the only FULL/complete version**, `OTHER_RESOURCES_TO_SORT/Tools/_DUMP_External_Utils_Lib/Scripts/chatgpt_parser.py` (+ a byte-identical `(2).py` dup) and `TheBigOne/archive/04_Utilities/Scripts/chatgpt_parser.py`, with a matching `test_chatgpt_parser.py` (13418 bytes, same 4 locations) containing a **synthetic but schema-accurate fixture** confirming `id`/`title`/`create_time`/`mapping`/`{message:{author:{role},content:{content_type,parts:[]},create_time}}` — exactly the shape already documented in `chatgpt_native_json`.

The full version's only genuinely new content is a **defensive multi-field-name schema pre-scan**
(`pre_scan_schema()`) that tries `id|conversation_id|uuid`, `title|name`, `create_time|created_at|timestamp`,
and `mapping|messages|turns`, plus `author|role|sender`/`content|text|parts` inside messages — untested
speculative field-name fallbacks, not a confirmed second real ChatGPT export shape. **Folded into
`chatgpt_native_json`'s `parsing_gotchas`** rather than added as a separate variant, since it doesn't
represent a distinct confirmed format.

**New variant added: `chatgpt_share_md`** — ChatGPT's built-in "Share" link markdown export (`**User:**
Name (email)` header, `Created:/Updated:/Exported:` timestamps, a `chatgpt.com/c/<uuid>` link,
`**Response:**`/`**ChatGPT said:**` turn markers) — mined from the in-repo `chatgpt_share.py`, a THIRD
distinct ChatGPT markdown shape alongside the two already-verified `chatgpt_md` and `customgpt_md`
variants. Not found as a standalone artifact anywhere in donor dumps.

### Gemini — the Chrome-extension exporter, found

**New variant added: `gemini_chrome_extension_md`** — this is the owner-flagged Chrome-extension exporter.
Mined from `server/vendored/chatminer/parsers/gemini_chrome.py`, whose docstring states the format was
taken "from your actual files" and names the exact filename convention `Google_Gemini_YYYY-MM-DD_HHMM.md`.
Structurally distinct from the already-verified `gemini_md` (native download, `**You:**`/`**Gemini:**`
bold markers, `Gemini - <title>.md` filename): the Chrome-extension export uses a `Google Gemini` /
`Conversation Details` / `Exported on: ... | Total Messages: N` header block and **emoji role markers**
(🤖 Assistant / 👤 You). Treated as high-confidence-mined (real owner samples per the code's own docstring)
rather than freshly re-verified, since this pass did not re-open a raw `.md` sample.

**New variant added: `gemini_json`** — a third, speculative Gemini export family (`{"messages":[{"author",
"content","timestamp"}]}`, speculated Google Takeout origin), mined from `gemini_json.py`. Its own docstring
does not claim real-sample provenance — kept explicitly unverified/speculative, distinct in confidence
from `gemini_chrome_extension_md`.

Neither new Gemini variant was found in the DuckDB donor-dump index or `dev-resources`/`_stale` — both
exist only as in-repo code.

### Claude — flipped to verified via a real, currently-live sample on this box

**New variant added and empirically VERIFIED: `claude_code_session_jsonl`** — Claude Code CLI's real
session-log format. Verified by direct read-only inspection (structure/keys only, content redacted) of a
currently-active session file on this box:
`C:\Users\matts\.claude\projects\E--AI-Workspace-Projects-the-platform-workspace-Agno-MCP-Platform\2164047f-3475-4825-b044-c49ceaab099d.jsonl`.
One JSON event per line; a real session interleaves **14 distinct event types** (measured on a 4000-line
sample: `last-prompt` 79, `agent-setting` 88, `mode` 88, `permission-mode` 88, `attachment` 331,
`file-history-snapshot` 53, `user` 226, `assistant` 363, `system` 85, `queue-operation` 82,
`worktree-state` 67, `ai-title` 65, `agent-name` 65, `bridge-session` 7) — only `user`/`assistant` typed
events carry real conversation turns; everything else is session/tooling telemetry a naive parser must
filter out. Confirmed fields: `type`, `message.role`/`message.content` (string or typed content blocks),
`uuid`, `parentUuid`, `sessionId`, `timestamp` (ISO-8601 ms precision), `cwd`, `gitBranch`, `version`.

This shape traces to a **real prior-iteration artifact**: `server/tools/parsers/ai_chat/claude_code_jsonl.py`'s
docstring says it is a "rewrite of `extracted-code/parsers/chat-exports/ClaudeCodeJSONLParser`" — that
named prior artifact was located and opened: it is a zipped mirror of the real public GitHub repo
**`ClaudeCodeJSONLParser`** (a `combined-log-viewer.html` tool for exactly these session logs), present at
both `extracted-code/parsers/chat-exports/ClaudeCodeJSONLParser` and
`dev-resources/Archives/OTHER_RESOURCES_TO_SORT/utilities/parsers/chat-exports/ClaudeCodeJSONLParser`
(identical zip, no file extension — open as a zip archive, not a `.py` file) — independent third-party
confirmation of the same shape.

There is a second, simpler chatminer-vendored parser (`claude_code.py`, flat `{role,content,timestamp}`
JSONL with no telemetry events) that does **not** match what real Claude Code sessions look like on this
box — noted as a mismatch, not adopted as the verified shape.

Provider status flipped `unverified` → `verified` on the strength of this one variant (per the schema's
"at least one variant verified" rule). The three **claude.ai web-chat** variants (`claude_export_json`,
`claude_export_md`, `claude_projects_export`) remain individually unverified — Claude Code (CLI) and
claude.ai (web chat) are different products; no real claude.ai export sample was found anywhere. A second
independent in-repo implementation (`claude_ai_export.py`) assumes the same `chat_messages[]`/`sender`/
`text`/`created_at` shape already guessed in the descriptor, but its own docstring says "no prior extracted
parser existed" — two independent guesses agreeing is noted as mild corroboration, explicitly NOT treated
as proof.

### Perplexity — flipped to verified via two real named upstream OSS references

**New variant added: `perplexity_gdpr_json`** — Perplexity's official GDPR/privacy-export
`conversations.json` (`{"conversations":[{"uuid","title","updated_at","mode","collection_uuid","status",
"citations":[],"answers":[{"content","citations":[{"title","url"}],"created_at"}]}]}`). Mined from
`server/vendored/chatminer/parsers/perplexity_gdpr.py`, whose docstring states "Adapted from:
FraYoshi/perplexity-export-convert (Python 3.11)" — a real, named, publicly-identifiable GitHub converter
project. Notably, this shape has **no first_message_source** — only `answers[]` are modeled, no explicit
query/user-turn object exists in the JSON, a real structural gap flagged for any downstream ingest.

**New variant added: `perplexity_plugin_md`** — a browser-userscript export (H1 `# Query title` sections,
`## Sources` numbered `[Title](URL)` list, `## Related`/`## Follow-up` end marker). Mined from
`perplexity_plugin.py`, adapted from "greasyfork/Perplexity-AI-Chat-Exporter (userscript)" — a real,
named, publicly-identifiable Greasy Fork userscript. This is the Perplexity analogue of the owner's
"Chrome plugin" category, though technically a Tampermonkey/Greasemonkey userscript rather than a
packaged `.crx` extension (see caveat below).

Provider status flipped `unverified` → `verified` on the strength of these two named-upstream-sourced
variants (satisfies the task's "real samples/parsers proving the shape" bar), with an explicit caveat in
both variants and the provider notes: **neither was independently re-verified against a raw sample file
opened during this pass** — confidence rests on the real named upstream projects and the in-repo parsers'
fidelity to them, not on a byte-level sample inspected on this box. The original `perplexity_md` (now
understood, via the mined library, to be a deliberate low-confidence generic fallback — its own
`can_parse()` caps score at 0.7) and `perplexity_html` variants remain fully unverified.

## Chrome-extension exporters — searched for, mostly not found as raw artifacts

Per the owner's explicit flag ("Some are Chrome plugins... completely different than the next plugin"),
this pass searched the DuckDB index and grepped `dev-resources`/`_stale` for:
- `manifest.json` files near chat/gpt/gemini/claude/perplexity paths → **1 hit**, unrelated (a Claude Code
  plugin-marketplace template's `manifest.json`, not a chat exporter).
- `TypingMind`, `SaveGPT`, `*ChatGPT*Exporter*`, `*chat*exporter*` filenames → **0 hits**.
- `manifest_version`/`chrome.runtime.sendMessage`/`chrome.storage.local` JS/JSON content combined with
  gpt/gemini/claude/perplexity/chat keywords → **0 hits**.

**No raw Chrome-extension source (`.crx`, `manifest.json`, content-script `.js`) for any AI-chat exporter
was found anywhere on this box.** What *was* found is the **output shape** of two such tools, already
reverse-engineered into the current codebase:
- **`gemini_chrome_extension_md`** — genuinely a Chrome extension's markdown export (confirmed variant, see above).
- **`perplexity_plugin_md`** — a browser userscript (not a packaged extension, but the same
  "differently-shaped browser-tooling exporter" category).

If the owner has the actual extension/userscript source files (not just their output) stored somewhere
else, they were not located by this pass's search terms.

## Salvage recommendations for the stage-2 ingest parsers

1. **Wire `chatgpt_share.py`, `gemini_chrome.py`, `gemini_json.py`, `perplexity_gdpr.py`, `perplexity_plugin.py`
   into `cb_chat_dedup.py`'s `detect_md_format()`/`classify_file()`.** All five are real, working,
   registered `server/tools/parsers/ai_chat/*.py` atomic tools with their own detection heuristics
   (`can_parse()` confidence scoring) — but `cb_chat_dedup.py` (the Case Bible dedup/sort tool the
   descriptors ultimately serve) currently only recognizes `chatgpt_md`/`customgpt_md`/`dated_export_md`/
   `gemini_md`. Files in these five NEW shapes would currently be misclassified (most likely falling
   through to `dated_export_md` or `work_product`). **Named merge target:** `cb_chat_dedup.py`'s
   `detect_md_format()` precedence chain + a new JSON-shape branch for `perplexity_gdpr_json`.
2. **`ClaudeCodeJSONLParser` (real GitHub repo, zipped in two donor locations) has a
   `combined-log-viewer.html`** — worth extracting and inspecting for a second, independently-built
   Claude Code session viewer's event-type handling, as a cross-check against
   `server/tools/parsers/ai_chat/claude_code_jsonl.py`'s `_event_text()` filtering logic before trusting
   it as complete (14 event types were observed live on this box; confirm the viewer's own event
   allowlist matches or exceeds that).
3. **The "Sprint 1" `chatgpt_parser.py` full version's spaCy-based entity/artifact extraction
   (`extract_entities`/`extract_artifacts`, code-block regex + hash-based dedup)** is architecturally
   interesting as a *post-parse enrichment* pattern (separate from format detection) — not a salvage
   target for the chat-formats descriptors themselves, but worth a look if/when stage-2 ingest adds
   entity extraction over parsed chat transcripts.
4. **`perplexity_gdpr_json`'s missing first-message/query field** is a structural gap worth designing
   around now rather than discovering downstream: any stage-2 ingest treating "first user message" as a
   universal dedup key needs a Perplexity-specific fallback (title-only, or synthesized from the first
   answer's opening line).
5. **Do not adopt chatminer's `claude_code.py`** (the flat `{role,content,timestamp}` JSONL variant) as
   the primary Claude Code session-log detector — it does not match the real, live session shape found on
   this box (`claude_code_session_jsonl`, type/message-nested, 14 telemetry event types). It may be
   useful only for a hypothetical manually-flattened re-export, not raw CLI session files.

## Descriptor files updated

- `C:\Users\matts\.claude\local-plugins\plugins\case-bible\schemas\chat-formats\chatgpt.json` —
  added `chatgpt_share_md`; enriched `chatgpt_native_json` gotchas with the 4-version Sprint-1 lineage.
- `C:\Users\matts\.claude\local-plugins\plugins\case-bible\schemas\chat-formats\gemini.json` —
  added `gemini_chrome_extension_md` (the Chrome-extension exporter) and `gemini_json`.
- `C:\Users\matts\.claude\local-plugins\plugins\case-bible\schemas\chat-formats\claude.json` —
  added `claude_code_session_jsonl` (empirically verified); provider status `unverified` → `verified`;
  enriched `claude_export_json`/`claude_export_md` gotchas with corroborating-but-unproven evidence.
- `C:\Users\matts\.claude\local-plugins\plugins\case-bible\schemas\chat-formats\perplexity.json` —
  added `perplexity_gdpr_json` and `perplexity_plugin_md`; provider status `unverified` → `verified`;
  enriched `perplexity_md` gotchas.
- All four mirrored byte-identical to
  `C:\Users\matts\.claude\plugins\cache\casebible-local\case-bible\0.5.1\schemas\chat-formats\`.
- All four validated against `_descriptor.schema.json`'s required-key structure (top-level, per-variant,
  detection/structure/dedup_keys sub-objects) via an ad hoc Python schema-walk; global variant-id
  uniqueness across all 5 provider files (21 ids total) confirmed with zero duplicates.

No existing variants were removed. No git commit/push, no deploy, no code changes to any parser module.
