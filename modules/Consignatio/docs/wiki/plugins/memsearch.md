---
title: "memsearch"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# memsearch

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Local fork of memsearch (zilliztech) for Claude Code: semantic session memory + health check (verifies the memsearch CLI is the private fork build), guard and DuckDB result cleaner bundled. See UPSTREAM.md.

Source: `E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code`. Version: `0.4.18-local.16`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `memory-config`

Diagnose and configure MemSearch memory behavior for Claude Code and Codex. Use when the user asks about MemSearch configuration, plugin summarization, PROJECT.md/USER.md maintenance, memory directories, index health, provider routing, prompt files, or migration/compatibility questions.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/skills/memory-config/SKILL.md:1>) · SHA-256 `a06963dec4246e36ce031d16b5bc43bd6dadf3928ae3a6365c60bab00581ccf2`

### `memory-recall`

Search and recall relevant memories from past sessions via memsearch. Use when the user's question could benefit from historical context, past decisions, debugging notes, previous conversations, or project knowledge -- especially questions like 'what did I decide about X', 'why did we do Y', or 'have I seen this before'. Also use when the session-start memory context mentions related work. Typical flow: search for 3-5 chunks, expand the most relevant, optionally deep-drill into original transcripts via the anchor format. Skip when the question is purely about current code state (use Read/Grep), ephemeral (today's task only), or the user has explicitly asked to ignore memory.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/skills/memory-recall/SKILL.md:1>) · SHA-256 `175c184423a243a4cc2c3511a8e4bfe818ae072e117aafb55f1f0c14da67d715`

### `memory-to-skill`

Turn workflows from your MemSearch memory into reusable skills. Use when the user asks to make/create/extract/distill a skill from what they just did or from past work, review skill candidates, install a distilled skill, or 'turn this into a skill'. Manages MemSearch procedural-memory candidates under .memsearch/skill-candidates/, not the host agent's own skills system.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/skills/memory-to-skill/SKILL.md:1>) · SHA-256 `cd3061b589aad4cb1d47a7e00de9aa0880aa37d293c662a1c41974e63733ea66`

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `mcp/server.py`

memsearch MCP server: search, expand, recall and status over the one shared memsearch collection.

Hosted (Coolify app `memsearch-mcp` on ovh-files, beside Milvus) and served over streamable-HTTP for
ContextForge, which federates it as gateway `memsearch` and offers the tools to Claude Code and Codex
through one virtual server. The desktop keeps the journals, the watcher and the indexer (the owner's
local-indexer exception); this server only reads Milvus and embeds queries, so it never touches files.

Tools (all read-only):
  search  hybrid search (dense + BM25, the CLI's own search) with compact, cleaned results
  expand  the whole journal section around a chunk, rebuilt from the section's chunks in Milvus
  recall  search + expand, deduplicated and packed for an agent
  status  collection, row count, embedder, server and memsearch versions; optional embedding probe

Expand differs from `memsearch expand` in one way: the CLI reads the source file, this server joins the
section's stored chunks. Lines the ingest filter dropped before indexing (tool dumps, system tags) are
not in Milvus, so they are not in a rebuilt section either.

Configuration comes from the environment (Coolify app env; secrets never in git):
  MEMSEARCH_MILVUS_URI     default http://milvus:19530 (the memsearch-milvus container on the
                           shared `probata` network)
  MEMSEARCH_MILVUS_TOKEN   Milvus token (secret)
  MEMSEARCH_COLLECTION     default agent_session_memory_nemotron3
  MEMSEARCH_EMBED_PROVIDER default openai (an OpenAI-compatible endpoint)
  MEMSEARCH_EMBED_MODEL    default nvidia/nemotron-3-embed-1b
  MEMSEARCH_EMBED_BASE_URL default https://integrate.api.nvidia.com/v1
  NVIDIA_NIM_API_KEY       embedding key (secret)
  MCP_TRANSPORT            stdio (default) or http;  MCP_HOST, MCP_PORT for http

Byline: Claude Code · Opus 5.5 · 2026-10-02 (owner 03:31 and 03:33: an MCP surface for memsearch,
federated through ContextForge, the same tools in both apps).

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [server.py:1](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/mcp/server.py:1>) · SHA-256 `c654e18fc4493b12f894d274cea44e8e2ee72f7a63a75f9065ec707ed0cb7547`

### `scripts/codex_hook.py`

Run one memsearch hook for Codex: the same scripts Claude Code runs, windowless on Windows.

    pythonw.exe codex_hook.py <session-start|stop|session-end|provider-guard|health>

.codex-plugin/hooks.json calls this for every Codex hook event; hooks/hooks.json (Claude Code) calls
the same scripts directly. What this launcher adds for Codex, and why:

- MEMSEARCH_HARNESS=codex, so the shared scripts read the Codex transcript format, tag notes
  "Codex" and use the plugins.codex.* settings.
- MEMSEARCH_DIR, so both apps use the one shared store. Claude Code gets it from `env` in
  ~/.claude/settings.json. Codex has no hook environment setting, so this reads the value Codex
  gives its own shell, `shell_environment_policy.set.MEMSEARCH_DIR` in $CODEX_HOME/config.toml.
  The memory-recall skill runs in that shell, so hooks and skill see the same store.
- No console window: run with pythonw.exe (no console of its own); Git Bash starts with
  CREATE_NO_WINDOW, so every grandchild (memsearch, git, python) shares one hidden console.
- Stdout goes to a temp file, not a pipe: a detached child (the shared watcher, the Stop
  summarizer) would otherwise hold the pipe open and stall the hook until it timed out.
- It answers before the timeout in hooks.json. Codex kills a timed-out hook's whole process
  tree, which would take a freshly started watcher with it; a completed hook keeps its
  detached children.
- Exit code 2 is reported as 1: to Codex, exit 2 from a Stop hook means "block and continue the
  turn", which a crashed script must never trigger.
- One line per run in $MEMSEARCH_DIR/_health/hooks.log (time, event, exit, seconds, plugin root),
  with the stderr tail when a script fails.

Byline: Claude Code · Opus 5.5 · 2026-10-02. Replaces ~/.codex/hooks/memsearch_codex_hook.py
(Claude Code · Opus 5 · 2026-09-10), which ran the upstream zilliztech Codex hooks from a clone.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [codex_hook.py:1](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/scripts/codex_hook.py:1>) · SHA-256 `d82daf2e7adbba60b3275dec07b3530acc6b45ccf9065c3d38f4ff5ef4ad1be6`

### `scripts/codex_turn.py`

Read one turn of a Codex rollout for the Stop hook: the Codex twin of parse-transcript.sh
plus last_user_turn.py, which do the same for a Claude Code transcript.

The Codex Stop hook payload arrives on stdin. Output, on stdout:

    <turn_id>   <kind>  <scope>
    === Transcript of a conversation between User and Codex ===
    [User]: ...
    [Codex]: ...

- turn_id: the payload's `turn_id` (the rollout's `task_started` id for the same turn).
- kind: `junk` or `ok`, by the same rule and thresholds as last_user_turn.py: the user's text has
  no letters, or is 16+ characters drawn from four or fewer distinct ones; the assistant made no
  tool call; and its reply is under 400 characters. Pocket and stuck-key input is not summarized.
- scope: `main`, or `subagent` / `internal` for threads Codex starts itself (spawned agents,
  memory consolidation). Claude Code's Stop hook never fires for its subagents, so the Stop hook
  journals main threads only.
- The transcript lines follow parse-transcript.sh: the user's text and the assistant's text, with
  tool calls and tool output left out. When the rollout holds no usable turn, the second line is
  one of parse-transcript.sh's markers: (empty transcript), (no user message found), (empty turn).

Rollout shapes read (Codex 0.160): a turn starts at `event_msg/task_started` (alias
`turn_started`) carrying `turn_id`. The user's text is `event_msg/user_message` (legacy history
mode) or `event_msg/item_completed` with `item.type == "UserMessage"` (paginated mode); a
`response_item` user `message` is the fallback, skipping injected context (`<...>` blocks and
AGENTS.md instructions). The reply is the `response_item` assistant `message` text, plus the
payload's `last_assistant_message` when the rollout has not been flushed that far yet. A tool
call is any `response_item` whose type ends in `_call`.

    python3 codex_turn.py < payload.json
    python3 codex_turn.py --rollout <rollout.jsonl> [--turn-id ID] [--all]   # diagnosis

Byline: Claude Code · Opus 5.5 · 2026-10-02 (rollout notes from the 2026-09-30 attempt by
Claude Code · Sonnet 5.5, scripts/last_user_turn.py in the codex-skip-junk-turns worktree)

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [codex_turn.py:1](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/scripts/codex_turn.py:1>) · SHA-256 `253a63338a14aef7aa6b514f6e2a89294272d25da26cb9d7209a1e54c558fa37`

### `scripts/journal_drop_turn.py`

Remove the note(s) already written for one user turn from a memsearch daily journal.

The Stop hook can fire more than once for the same user turn (the turn continues after a
background task reports, for example). The later summary covers the whole turn, so the hook
calls this first and then appends the new note: one note per turn, never two.

    python3 journal_drop_turn.py <journal.md> <turn-uuid>

A note is the block from its `### HH:MM` heading to the next `##`/`###` heading. It is removed
when its anchor comment names the turn. Line endings are kept byte for byte. The journal is
shared by every session, so the file is rewritten only if it has not changed since it was
read, with a few retries.

Byline: Claude Code · Opus 5.5 · 2026-09-30

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [journal_drop_turn.py:1](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/scripts/journal_drop_turn.py:1>) · SHA-256 `c93aae5dc8270fa0c9ff4f88c78f7588c884f203285e7a584b6a4687a3184e5b`

### `scripts/last_user_turn.py`

Identify the last user turn of a Claude Code transcript for the Stop hook.

Prints `<uuid>  <kind>` for the last turn. `kind` is `junk` when the turn is pocket or
stuck-key input, and `ok` otherwise. A turn is junk only when all three hold:

- the user's text has no letters, or is 16+ characters drawn from four or fewer distinct ones;
- the assistant made no tool call in the turn;
- the assistant's reply is under 400 characters.

So a short real answer such as "1" or "B" that leads to work or to a discussion is kept.
The Stop hook skips junk turns: summarizing them filled the daily journal with empty notes
and, on 2026-09-30, one invented summary.

    python3 last_user_turn.py <transcript.jsonl>          # last turn
    python3 last_user_turn.py <transcript.jsonl> --all    # every turn, for diagnosis

Byline: Claude Code · Opus 5.5 · 2026-09-30

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [last_user_turn.py:1](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/scripts/last_user_turn.py:1>) · SHA-256 `8e01058a849cb30c10c473df5d7fe58f97593bc50584f300a6b94e51ec380b85`

### `scripts/maintenance-runner.py`

Plugin-local runner for MemSearch maintenance tasks.

This script belongs to the plugin layer. It handles host-native agent
invocations, while the Python package provides shared config, due-state,
prompt, and API-provider logic.

```text
python "E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/scripts/maintenance-runner.py" --help
```

Declared arguments: `--force`, `--json-output`, `--memsearch-dir`, `--platform`, `--project-dir`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --platform | — | True | ['claude-code', 'codex', 'opencode', 'openclaw', 'dsh'] | — |
| --project-dir | — | False | — | — |
| --memsearch-dir | — | False | — | — |
| --force | store_true | False | — | — |
| --json-output | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [maintenance-runner.py:1](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/scripts/maintenance-runner.py:1>) · SHA-256 `9f9ee53afd8017c2efd0be4293b4747282a9c8e1532e4b62ac3d97e8b68ed72c`

### `scripts/memsearch_clean.py`

memsearch_clean - DuckDB filter that makes memsearch search/expand output context-efficient.

Byline: Claude Code · Opus 5 · 2026-09-14

Pipe JSON from memsearch into it:
  memsearch search "<q>" --top-k 8 -j --collection C | python3 ${CLAUDE_PLUGIN_ROOT}/scripts/memsearch_clean.py
  memsearch expand <hash> -j --collection C         | python3 ${CLAUDE_PLUGIN_ROOT}/scripts/memsearch_clean.py
Options: --max-chars N (per result, default 700; 0 = no trim)   --top N (search: keep best N, default all)

What it removes (all in DuckDB SQL):
  - CRLF, <!-- anchor --> comments (search output only; expand keeps ONE compact "ref:" line with
    session/turn/transcript so the transcript drill-down still works)
  - <system-reminder>, <shell_metadata>, <task-notification>, <command-*> blocks
  - "[Tool: ...]" lines AND the tool output under them (until the next [User]/[Assistant]/heading/bullet)
  - "- Memory summary unavailable: ..." stubs, bare "### HH:MM" time headings, blank-line runs
  - results that are empty after cleaning, and duplicates (same normalized text -> keep best score)
Ingest-side twin: memsearch/ingest_filter.py in the memsearch fork (~/.claude/local-plugins/forks/memsearch; shim patch 3 until 2026-09-26) applies the
same line rules before chunks are embedded, so new noise is not indexed in the first place.

```text
python "E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/scripts/memsearch_clean.py" --help
```

Declared arguments: `--max-chars`, `--top`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --max-chars | int | False | — | — |
| --top | int | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [memsearch_clean.py:1](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/scripts/memsearch_clean.py:1>) · SHA-256 `1cad6816408237c02951f8f26c9c59a0f0e03e514603dc3e25d6ac52c6bdb2bf`

### `scripts/memsearch_health.py`

memsearch health check - SessionStart hook + manual CLI.

Byline: Claude Code · Opus 5 · 2026-09-14
Amended: Claude Code · Opus 5.5 · 2026-09-26 - shim check replaced by a fork-build check; failures also
reach the agent as SessionStart additionalContext.

Validates that memsearch (~/.memsearch, Stop-hook summaries + Milvus index) is actually FUNCTIONING,
using live calls, not config inspection. memsearch's own hooks discard stderr, so embed/summarize
failures are otherwise silent until someone notices stub summaries or a stale index.

Checks (network ones run in parallel, each hard-capped):
  embed      real POST /embeddings (2 texts) against [embedding] - must return 2 vectors
  summarizer real POST /chat/completions against the provider in [plugins.<agent>.summarize]
             (claude-code, or codex when MEMSEARCH_HARNESS=codex)
             (thinking disabled for nvidia/nemotron-3*, as the memsearch fork does); slow = warn.
             On failure, probes the other configured providers and names a working backup.
  milvus     collection exists + rowCount via Milvus REST v2
  fork       the memsearch CLI is the Propria fork build (0.4.x+propriaN) with its NIM input guard live;
             ~~shim: NIM no-think shim present in the memsearch venv~~ (shim retired 2026-09-26)
  index      ~/.memsearch/.index-state.json status (degraded/error/stuck) + failed files
  stubs      NEW "Memory summary unavailable" lines in memory/*.md since the previous run

Output: hook mode -> one {"systemMessage": ...} line (short OK line, or a warning block); on failure the
        block is also sent to the agent as additionalContext.
        --check  -> plain text, exit 1 if anything failed.
Last report: ~/.memsearch/_health/last.txt (state for stub tracking: state.json).
Context: memory note memsearch-summarizer-nemotron-shim.md (2026-09-13 lightning outage).

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [memsearch_health.py:1](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/scripts/memsearch_health.py:1>) · SHA-256 `f18d4cfabd0b374633b82fb17e21d8913aa4202e7a617caeb2f3485ffaba944e`

### `scripts/memsearch_provider_guard.py`

memsearch provider guard — SessionStart hook.

Byline: Claude Code · Fable 5 · 2026-08-26

~/.memsearch/config.toml's [embedding] provider has been silently flipped from "openai" to
"onnx" twice (2026-08-23 and 2026-08-26 17:24). "onnx" resolves the NIM model id as a gated
HuggingFace repo -> 401 -> every memsearch search/index dies. This hook restores the [embedding]
block from ~/.memsearch/_pinned/config.toml.pinned when provider != openai, backs the bad file
up first (never deletes), and reports via systemMessage. Same pattern as ccc_patch_guard.py.
Re-pin after any DELIBERATE change:  cp ~/.memsearch/config.toml ~/.memsearch/_pinned/config.toml.pinned

~~2026-09-06 (Fable 5.1): also re-installs the NIM no-think shim (memsearch_nim_nothink.{py,pth}) into the
memsearch uv-tool venv if a reinstall dropped it. Without it, nvidia/nemotron-3.5-lightning-30b-a3b
thinks for ~70 s+ per summary and every Stop-hook summary becomes a "summary unavailable" stub.
Canonical copy + install.sh: scripts/memsearch_nim_nothink/ in this plugin (moved from ~/.claude/hooks 2026-09-14)~~
Corrected 2026-09-26 (Claude Code · Opus 5.5): the shim is retired. Its fixes live in the memsearch
application itself, the private fork at ~/.claude/local-plugins/forks/memsearch, so this guard no longer
touches the venv. memsearch_health.py checks that the installed CLI is the fork build.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [memsearch_provider_guard.py:1](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/scripts/memsearch_provider_guard.py:1>) · SHA-256 `0542396f058edada5f0951292b386c75711b3caaa69c64f8fdde4c6171520712`

### `transcript.py`

Parse Claude Code JSONL transcripts for progressive memory disclosure.

```text
python "E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/transcript.py" --help
```

Declared arguments: `--context`, `--json-output`, `--turn`, `-c`, `-j`, `-t`, `jsonl_path`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| jsonl_path | — | positional | — | Path to the JSONL transcript file. |
| --turn, -t | — | False | — | Target turn UUID (prefix match). |
| --context, -c | int | False | — | Number of turns before/after target. |
| --json-output, -j | store_true | False | — | Output as JSON. |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [transcript.py:1](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/transcript.py:1>) · SHA-256 `a587c3a4a4a8943efd931240154c675b165ca3aeda90e02e0c3102cadd7607a6`

## Mcp Tools

### `search`

Search the shared memsearch memory (hybrid dense + BM25, the same search as `memsearch search`).

query: what to look for, in natural language or exact words.
top_k: results to return (1-20).
source_contains: optional substring the source path must contain, e.g. "2026-10-02" for one day's
    journal or "memory\codex" for the imported Codex memories.
max_chars: per-result text limit (0 = no limit).
Each hit has chunk_hash (for expand), file, date, heading, agent, score and cleaned text.
A query none of whose words occurs in the collection returns no hits.

| Parameter | Declared type |
|---|---|
| `query` | str |
| `top_k` | int |
| `source_contains` | str |
| `max_chars` | int |

Validation: source declaration; invocation not tested.

Source: [server.py:238](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/mcp/server.py:238>) · SHA-256 `c654e18fc4493b12f894d274cea44e8e2ee72f7a63a75f9065ec707ed0cb7547`

### `expand`

The whole journal section (one note) around a chunk, rebuilt from its chunks in Milvus.

chunk_hash: from a search hit.
max_chars: text limit (0 = no limit).
Returns file, date, heading, agent, line range, the note's anchor (session, turn, transcript path)
and the cleaned text. Lines the ingest filter dropped before indexing are not stored, so they are
not shown.

| Parameter | Declared type |
|---|---|
| `chunk_hash` | str |
| `max_chars` | int |

Validation: source declaration; invocation not tested.

Source: [server.py:270](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/mcp/server.py:270>) · SHA-256 `c654e18fc4493b12f894d274cea44e8e2ee72f7a63a75f9065ec707ed0cb7547`

### `recall`

Search, expand the best notes and pack them for an agent: the memory answer in one call.

query: what to recall.
top_k: notes to include (1-10).
max_chars: total text budget (default 8000).
Each note is headed "[n] date · heading · agent · score".

| Parameter | Declared type |
|---|---|
| `query` | str |
| `top_k` | int |
| `max_chars` | int |

Validation: source declaration; invocation not tested.

Source: [server.py:289](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/mcp/server.py:289>) · SHA-256 `c654e18fc4493b12f894d274cea44e8e2ee72f7a63a75f9065ec707ed0cb7547`

### `status`

memsearch server health: collection, row count, embedder, versions.

probe_embedding: also embed one short text to prove the embedding endpoint answers.

| Parameter | Declared type |
|---|---|
| `probe_embedding` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:323](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/mcp/server.py:323>) · SHA-256 `c654e18fc4493b12f894d274cea44e8e2ee72f7a63a75f9065ec707ed0cb7547`

## Mcp Servers

### `memsearch`

http

Validation: configured; health not inferred.

Source: [.mcp.json:1](<E:/AI_Workspace/plugins/forks/memsearch/plugins/claude-code/.mcp.json:1>) · SHA-256 `ba55383c1bc312bb48b95a81dfc26f10513a7074f111fde2893df8df50a69829`


## Existing hosted MCP exposure

The live ContextForge server associated with this plugin exposes the following names. Complete argument schemas and descriptions are in [[Code/wiki/contextforge-tools]]. This is registry discovery, not invocation proof.

- `memsearch-expand`
- `memsearch-recall`
- `memsearch-search`
- `memsearch-status`

Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
