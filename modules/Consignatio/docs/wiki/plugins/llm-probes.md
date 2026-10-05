---
title: "llm-probes"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# llm-probes

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Live model probing: which NIM / OpenRouter / other provider models work right now for chat, tools, vision, embeddings, rerank; NIM probes (incl. streaming + max-length/rerank rechecks) kept separate from a provider-agnostic probe driven by providers.json; the llm-probe service client and TUI. 5 member skills (`llm-probes:<name>`) plus the index entry skill.

Source: `E:/AI_Workspace/plugins/plugins/llm-probes`. Version: `1.2.1`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `lp-llm-probe`

(llm-probes) Client for the llm-probe service (FastAPI, deployed on ovh-files 100.91.190.107:8030, tailnet-only) that live-tests any model across 8 providers (NIM, Ollama Cloud, OpenRouter, Google, OpenAI, Mistral, Groq, Cerebras) — liveness, tool-calling, one-sentence summarization, exact-format instruction-following, or a free-form playground prompt with full max_tokens/temperature/reasoning_effort control. Every result persists to casebible.llm_eval in Postgres. Use when asked to test/probe/rerun a specific model, check if a provider is live, compare models across providers, investigate why a model fails a probe (reasoning-token overhead, wrong prompt shape, etc.), or browse/summarize past results. Also launches an interactive TUI (`llm-probe tui`) for a human to drive the same thing.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/llm-probes/skills/llm-probe/SKILL.md:1>) · SHA-256 `8ad9b2de8ce4ab5e28bcfa9415a8a13fb21040537660352969d442d662403d9d`

### `llm-probes`

(llm-probes) Live model probing: which NIM / OpenRouter / other provider models work right now for chat, tools, vision, embeddings, rerank; provider-agnostic probe from providers.json; the llm-probe service client and TUI. Entry point / router — read this first, then load one member from references/. Triggers: nim, nvidia, openrouter, free models, embedder, which models work, is model retired, probe, llm-probe. Members: nim-chat-probe, nim-embed-probe, openrouter-free-probe, llm-probe.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/llm-probes/skills/llm-probes/SKILL.md:1>) · SHA-256 `87fa1a71d26ffab1b276a53cbfe318dce89a0b2001325099bdf92d394b91b6ed`

### `lp-nim-chat-probe`

(llm-probes) Live-test every NVIDIA NIM chat/text model (integrate.api.nvidia.com) for plain chat, tool calling, and vision (image input), recording latency + prompt/completion token usage per call. Lists /models, skips embedders, runs the checks concurrently, and saves every run to runs/ with a diff vs the previous run. Any single check can be run alone via --tests chat|tools|vision. A streaming recheck re-tests ambiguous timeouts so slow reasoning models are not misreported as dead. Use when asked 'which NIM chat models work', 'which NIM models support tool calling / function calling', 'which NIM models are vision-capable / take images', 'how fast is MODEL on NIM', 'is MODEL retired', or before picking a chat model for an agent, Portkey route, or n8n workflow.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/llm-probes/skills/nim-chat-probe/SKILL.md:1>) · SHA-256 `3da48c41d7c0f9abd09fd95fb4e720bd967fc9d3dc5d176b56b5ece104484c1f`

### `lp-nim-embed-probe`

(llm-probes) Live-test which NVIDIA NIM embedding AND reranking models actually work right now. Lists /models, filters embedders, POSTs a real 4-text batch to each (retrying with input_type=passage for asymmetric models), always re-probes known/retired ids so EOLs are caught, and saves every run to runs/ with a diff vs the previous run. The max-length recheck finds each model's real token ceiling and probes rerankers on their retrieval endpoints (rerankers are NOT in /models). Use when asked 'what embedders or rerankers are available on NIM', 'is MODEL still up', 'did NIM retire X', before picking/changing an embed or rerank model, or when an embed/rerank pipeline suddenly returns 404/410/empty results.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/llm-probes/skills/nim-embed-probe/SKILL.md:1>) · SHA-256 `e167f5f6f2e7aff999dd08b622331ececdd7d8ee41902a525206245031339e9e`

### `lp-openrouter-free-probe`

(llm-probes) Live-test every OpenRouter FREE model (':free' ids / zero-priced) for plain chat, tool calling, and vision (image input), recording latency, prompt/completion token usage, and OpenRouter's own capability claims (input_modalities, supported_parameters, context) next to the live result. Saves every run to runs/ with a diff vs the previous run; any single check can run alone via --tests chat|tools|vision. Use when asked 'which OpenRouter free models work', 'free models with tool calling / vision', 'is MODEL:free still up / rate-limited', or before routing a Portkey fallback chain, n8n workflow, or agent to a zero-cost model.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/llm-probes/skills/openrouter-free-probe/SKILL.md:1>) · SHA-256 `27277cd4d5a20e6bb601b81a6caba0d8a8af0f1474e76d0f5dbe8d00a3d44f7e`

### `lp-provider-probe`

(llm-probes) Provider-agnostic live model probe driven by providers.json, a provider list we maintain. Pick a provider with --provider or the PROBE_PROVIDER variable and it tests that provider's chat (SSE streaming, first-token timing, PONG check), embeddings (dimension + paraphrase sanity) and rerank (ordering sanity), saving every run with a diff against the previous one. Use when asked 'which Groq / Cerebras / Mistral / OpenAI / Gemini / Voyage / OpenRouter / Ollama Cloud models work right now', when adding a new provider to probe, or before wiring a provider into a gateway or pipeline. NVIDIA NIM is deliberately separate: use nim-chat-probe / nim-embed-probe.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/llm-probes/skills/provider-probe/SKILL.md:1>) · SHA-256 `ac3399e8058203d41f3c5cb13a7552393cc5a2c99e81211016f3bdcf4581a216`

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `skills/llm-probe/scripts/llm_probe_cli/cli.py`

llm-probe CLI — scriptable client for the llm-probe service (deployed on
ovh-files, tailnet-only, http://100.91.190.107:8030 by default; override
with LLM_PROBE_URL). Every subcommand also takes --json for machine-readable
output — this is what the llm-probe SKILL.md tells agents to call.

`llm-probe tui` launches the interactive Textual app (llm_probe_cli.tui) for
human use instead of one-shot commands.

Byline: Claude Code · Sonnet 5 · 2026-08-27

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cli.py:1](<E:/AI_Workspace/plugins/plugins/llm-probes/skills/llm-probe/scripts/llm_probe_cli/cli.py:1>) · SHA-256 `7cc9b45a75a6154f9f9edab63368c1a285345e65bad4e8eaceb31537e9c813e7`

### `skills/llm-probe/scripts/llm_probe_cli/tui.py`

llm-probe TUI — interactive terminal client for the llm-probe service.
Three tabs: Playground (compose a prompt, pick provider/model/params, run it
live), Board (the full liveness+capability grid), History (past playground
runs). All network calls run in a worker thread so the UI never blocks.

Run via: python -m llm_probe_cli.cli tui   (or directly: python -m llm_probe_cli.tui)

Byline: Claude Code · Sonnet 5 · 2026-08-27

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [tui.py:1](<E:/AI_Workspace/plugins/plugins/llm-probes/skills/llm-probe/scripts/llm_probe_cli/tui.py:1>) · SHA-256 `d5b2278a9c50b09b93a9e85dbbbc8bb80525705b6b3bde6fb78e898302e0ae63`

### `skills/nim-chat-probe/scripts/nim_chat_probe.py`

nim-chat-probe — LIVE-test every NVIDIA NIM chat/text model: plain chat, tool calling,
vision (image input), with latency + token usage per call.

Byline: Claude Code · Fable 5 · 2026-08-26

Usage:
  python nim_chat_probe.py                     # probe all non-embedding models, save run, diff vs previous
  python nim_chat_probe.py --only a/b c/d      # probe just these ids
  python nim_chat_probe.py --extra a/b         # probe extra ids not in /models
  python nim_chat_probe.py --workers 8         # concurrency (default 6)
  python nim_chat_probe.py --tests vision      # run only one/some checks: chat,tools,vision (default all)
  python nim_chat_probe.py --last              # print newest saved report (no API calls)
  python nim_chat_probe.py --history           # one line per saved run

Per model, three real calls to /chat/completions:
  chat   : "Reply with exactly the word PONG."            -> latency, prompt/completion tokens
  tools  : a get_weather tool + "What's the weather in Paris?" -> did it emit a tool_call?
  vision : 1x1 PNG data-URI + "What color is this image?"  -> accepted image content parts?
Key: env NVIDIA_API_KEY (User-scope registry var) or NVIDIA_NIM_API_KEY.
Runs persist in ../runs/<UTC-stamp>.json + latest.md for later recall.

```text
python "E:/AI_Workspace/plugins/plugins/llm-probes/skills/nim-chat-probe/scripts/nim_chat_probe.py" --help
```

Declared arguments: `--extra`, `--history`, `--last`, `--only`, `--tests`, `--workers`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --only | — | False | — | — |
| --extra | — | False | — | — |
| --workers | int | False | — | — |
| --tests | — | False | — | comma list of chat,tools,vision |
| --last | store_true | False | — | — |
| --history | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [nim_chat_probe.py:1](<E:/AI_Workspace/plugins/plugins/llm-probes/skills/nim-chat-probe/scripts/nim_chat_probe.py:1>) · SHA-256 `6bfd91dad4e853a41971a83b8e9267ad731cc9ef1c0c28f01da3e6a90900e9ce`

### `skills/nim-chat-probe/scripts/nim_streaming_recheck.py`

Recheck ambiguous NIM chat results using SSE streaming without a post-first-token timeout.

```text
python "E:/AI_Workspace/plugins/plugins/llm-probes/skills/nim-chat-probe/scripts/nim_streaming_recheck.py" --help
```

Declared arguments: `--first-event-timeout`, `--only`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --only | — | False | — | Run only these catalog model IDs |
| --first-event-timeout | float | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [nim_streaming_recheck.py:1](<E:/AI_Workspace/plugins/plugins/llm-probes/skills/nim-chat-probe/scripts/nim_streaming_recheck.py:1>) · SHA-256 `3d0108f3d41503944185de91a6d68c7978c5353b8ea66a4ab855e46eee8077a5`

### `skills/nim-embed-probe/scripts/nim_embed_probe.py`

nim-embed-probe — list NVIDIA NIM embedding models and LIVE-test each one.

Byline: Claude Code · Fable 5 · 2026-08-26

Usage:
  python nim_embed_probe.py                # list + probe, save run, diff vs previous
  python nim_embed_probe.py --extra a/b    # also probe models not in /models (retired?)
  python nim_embed_probe.py --last         # just print the most recent saved run (no API calls)
  python nim_embed_probe.py --history      # one line per saved run

Key comes from env NVIDIA_API_KEY (User-scope registry var on this box) or NVIDIA_NIM_API_KEY.
Runs are saved next to this skill in ../runs/<UTC-stamp>.json + latest.md so any later session
can recall "what was working on <date>" without re-hitting the API.

```text
python "E:/AI_Workspace/plugins/plugins/llm-probes/skills/nim-embed-probe/scripts/nim_embed_probe.py" --help
```

Declared arguments: `--extra`, `--history`, `--last`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --extra | — | False | — | extra model ids to probe |
| --last | store_true | False | — | — |
| --history | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [nim_embed_probe.py:1](<E:/AI_Workspace/plugins/plugins/llm-probes/skills/nim-embed-probe/scripts/nim_embed_probe.py:1>) · SHA-256 `f22ea20d2f5fe64b92591f77a7f45a932fc3f7d4c7bee24a2e1b21436a4403e7`

### `skills/nim-embed-probe/scripts/nim_embed_rerank_recheck.py`

Recheck NIM embedding and reranking models with max-length inputs and a generous timeout.

Embed/rerank copy of nim_streaming_recheck.py (which excludes these models from its chat list).

> _Byline: Claude Code · Opus 5 · 2026-09-10_

Per model, three requests:
  1. short   - 4 texts (embed) / 4 passages (rerank): dimension, batching, semantic sanity.
  2. over    - one over-long input with truncate=NONE: the 4xx names the real token ceiling.
  3. long    - a batch of long inputs with truncate=END: the model working AT its ceiling.

Limits observed live 2026-09-10: nemotron-3-embed-1b 4096 tokens + 65536 chars per input;
llama-nemotron-rerank-vl-1b-v2 10240 tokens per query+passage.

```text
python "E:/AI_Workspace/plugins/plugins/llm-probes/skills/nim-embed-probe/scripts/nim_embed_rerank_recheck.py" --help
```

Declared arguments: `--batch`, `--long-chars`, `--only`, `--passages`, `--rerank-long-chars`, `--rerankers`, `--skip-embed`, `--skip-rerank`, `--timeout`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --only | — | False | — | Embed model IDs to probe (default: catalog embedders) |
| --rerankers | — | False | — | Reranker model IDs to probe |
| --skip-embed | store_true | False | — | — |
| --skip-rerank | store_true | False | — | — |
| --timeout | float | False | — | Per-request read timeout in seconds |
| --long-chars | int | False | — | Chars per long embed input (NIM caps embed inputs at 65536 chars) |
| --batch | int | False | — | Long inputs per embed batch request |
| --rerank-long-chars | int | False | — | Chars per long rerank passage (~14k tokens, over the 10240 ceiling) |
| --passages | int | False | — | Passages in the long rerank request |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [nim_embed_rerank_recheck.py:1](<E:/AI_Workspace/plugins/plugins/llm-probes/skills/nim-embed-probe/scripts/nim_embed_rerank_recheck.py:1>) · SHA-256 `842b50b34bbe01f1c2a96b8a4dae8cb04344a6dba0c0888ef9139554abea4868`

### `skills/openrouter-free-probe/scripts/openrouter_free_probe.py`

openrouter-free-probe — LIVE-test every OpenRouter FREE model: plain chat, tool calling,
vision (image input), with latency + token usage per call. Sister of nim-chat-probe.

Byline: Claude Code · Fable 5 · 2026-08-26

Usage:
  python openrouter_free_probe.py                    # all free models, all 3 checks, save + diff
  python openrouter_free_probe.py --tests tools      # one/some checks only: chat,tools,vision
  python openrouter_free_probe.py --only a/b:free    # specific ids
  python openrouter_free_probe.py --extra a/b        # extra ids (need not be free)
  python openrouter_free_probe.py --workers 3        # concurrency (default 2 — free tier is ~20 req/min)
  python openrouter_free_probe.py --last | --history

"Free" = id ends with ":free" OR pricing.prompt == pricing.completion == "0".
/models also tells us what OpenRouter *claims* (input_modalities, supported_parameters); we
record the claim next to the live result so claim-vs-reality drift is visible.
Key: env OPENROUTER_API_KEY. Runs persist in ../runs/<UTC-stamp>.json + latest.md.

```text
python "E:/AI_Workspace/plugins/plugins/llm-probes/skills/openrouter-free-probe/scripts/openrouter_free_probe.py" --help
```

Declared arguments: `--extra`, `--history`, `--last`, `--only`, `--tests`, `--workers`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --only | — | False | — | — |
| --extra | — | False | — | — |
| --workers | int | False | — | — |
| --tests | — | False | — | — |
| --last | store_true | False | — | — |
| --history | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [openrouter_free_probe.py:1](<E:/AI_Workspace/plugins/plugins/llm-probes/skills/openrouter-free-probe/scripts/openrouter_free_probe.py:1>) · SHA-256 `d997fae3666a3496a253437e339198385d2ff6c8f935ffa21c3bcf2967e693c3`

### `skills/provider-probe/scripts/provider_probe.py`

provider-probe — live-test any provider listed in providers.json: chat, embeddings, rerank.

Byline: Claude Code · Opus 5 · 2026-09-10

Owner orders 2026-09-10: make a plugin out of the NIM probe files; make one provider-agnostic, accepting a
variable, from a list we create; keep NIM separate. NVIDIA NIM is deliberately NOT in providers.json — use
nim-chat-probe / nim-embed-probe. The chat check uses the method from nim_streaming_recheck.py: only the
first streamed event has a timeout, then the stream is read to [DONE], so slow reasoning models are not
misreported as dead.

Usage:
  python provider_probe.py --list                                  # providers + whether each key is present
  python provider_probe.py --provider groq                         # catalog models, the provider's kinds
  PROBE_PROVIDER=voyage python provider_probe.py                   # provider chosen by the variable
  python provider_probe.py --provider openrouter --match :free --kinds chat --limit 10
  python provider_probe.py --provider mistral --only mistral-embed --kinds embed
  python provider_probe.py --provider groq --last                  # newest saved report, no API calls
  python provider_probe.py --provider groq --history               # one line per saved run

Keys: the provider's key_env from the environment, else ~/.secrets/*.env (parsed, never sourced, never printed).
Runs: ../runs/<provider>/<UTC>.json + latest.md, with a diff against the previous run.
Providers file: ../providers.json, or PROBE_PROVIDERS_FILE.

```text
python "E:/AI_Workspace/plugins/plugins/llm-probes/skills/provider-probe/scripts/provider_probe.py" --help
```

Declared arguments: `--first-event-timeout`, `--history`, `--kinds`, `--last`, `--limit`, `--list`, `--match`, `--only`, `--provider`, `--workers`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --provider | — | False | — | provider name from providers.json (or PROBE_PROVIDER) |
| --list | store_true | False | — | list providers and whether each key is present |
| --kinds | — | False | — | comma list of chat,embed,rerank (default: the provider's kinds) |
| --only | — | False | — | probe only these model ids |
| --match | — | False | — | only model ids containing this text |
| --limit | int | False | — | max models per kind (0 = all) |
| --workers | int | False | — | — |
| --first-event-timeout | float | False | — | — |
| --last | store_true | False | — | — |
| --history | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [provider_probe.py:1](<E:/AI_Workspace/plugins/plugins/llm-probes/skills/provider-probe/scripts/provider_probe.py:1>) · SHA-256 `296d426ee3fc6c1ac7a90ebfc56371c90a95dbc88696e8c7cf426e96de662a4a`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
