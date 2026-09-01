---
name: llm-probe
description: Client for the llm-probe service (FastAPI, deployed on ovh-files 100.91.190.107:8030, tailnet-only) that live-tests any model across 8 providers (NIM, Ollama Cloud, OpenRouter, Google, OpenAI, Mistral, Groq, Cerebras) — liveness, tool-calling, one-sentence summarization, exact-format instruction-following, or a free-form playground prompt with full max_tokens/temperature/reasoning_effort control. Every result persists to casebible.llm_eval in Postgres. Use when asked to test/probe/rerun a specific model, check if a provider is live, compare models across providers, investigate why a model fails a probe (reasoning-token overhead, wrong prompt shape, etc.), or browse/summarize past results. Also launches an interactive TUI (`llm-probe tui`) for a human to drive the same thing.
---

# llm-probe

> _Byline: Claude Code · Sonnet 5 · 2026-08-27_

Thin client — all the actual provider calls and scoring happen server-side in
`Agno-MCP-Platform/llm_probe/` (source of truth for prompts/scoring logic).
This skill just talks to it over the tailnet. Base URL defaults to
`http://100.91.190.107:8030`; override with `LLM_PROBE_URL` if the service
ever moves.

```
python -m llm_probe_cli.cli health                                  # is it up, which providers have keys
python -m llm_probe_cli.cli providers                                # list providers + configured status
python -m llm_probe_cli.cli models nim                               # live catalog fetch for one provider
python -m llm_probe_cli.cli probes                                   # the 4 named probes + their exact prompts

python -m llm_probe_cli.cli probe nim nvidia/nemotron-3-super-120b-a12b liveness
python -m llm_probe_cli.cli probe google gemini-2.5-flash summarization --reasoning-effort none

python -m llm_probe_cli.cli run google gemini-2.5-flash "your prompt here" \
    --max-tokens 500 --reasoning-effort none          # free-form playground call, persisted

python -m llm_probe_cli.cli board --live-only                        # the full liveness+capability grid
python -m llm_probe_cli.cli board --untested-only                    # live models with no Tier-1 result yet
python -m llm_probe_cli.cli summary                                  # pass/fail tallies across everything
python -m llm_probe_cli.cli history --limit 20                       # recent playground runs

python -m llm_probe_cli.cli tui                                      # interactive terminal app for a human
```

Every subcommand takes `--json` for machine-readable output (what you want
when chaining/parsing). Run from `~/.claude/skills/llm-probe/scripts/` (or
`cd` there first) so `llm_probe_cli` resolves as a package —
`python -m llm_probe_cli.cli <command>`.

**The `reasoning_effort` fix**: several models (Google's Gemini 3.x line
especially, also assorted NIM/Ollama Cloud/OpenRouter reasoning models) burn
part of `max_tokens` on a hidden "thinking" pass before the visible answer,
which reads as a truncated/empty response if the budget wasn't sized for it.
`--reasoning-effort none` on `probe`/`run` eliminates that overhead entirely
on providers that support it (verified live on Google — 230 hidden tokens →
0). Harmless no-op on providers without the concept. `run`'s output line
shows `hidden reasoning tokens: N` whenever `usage.total_tokens` exceeds
`completion + prompt` — that delta IS the thinking cost, since providers
don't expose the trace text itself (Google never does; some OpenRouter/Ollama
reasoning models leak it inline in `content` instead — you'll just see it).

**Named probes vs `run`**: the four named probes (`liveness`,
`tool_use`, `summarization`, `instruction_following`) use fixed prompts and
get scored pass/fail — use these to reproduce/extend the standing benchmark
in `casebible.llm_eval`. `run` is unscored — use it to iterate on a NEW
prompt/task before deciding it's worth turning into a named probe, or just to
manually sanity-check a model.

**Persistence**: everything writes to Postgres (`casebible.llm_eval` —
`probe_run`/`probe_result` for named probes, `playground_run` for `run`
calls) unless you pass `--no-persist`. The liveboard artifact and `board`/
`summary` here read the same live data — no separate export step.

Server source + deploy config: `Agno-MCP-Platform/llm_probe/` +
`Agno-MCP-Platform/deploy/llm-probe.yaml`. To change a probe's prompt or
scoring, edit `llm_probe/probes.py` there and redeploy — this skill has no
opinion on what a probe tests, only how to call it.

**Relationship to nim-chat-probe / nim-embed-probe / openrouter-free-probe**:
those three run locally, hit one provider each, and save JSON to their own
`runs/` dir — no cross-provider view, no persistence, no scoring beyond raw
liveness. This skill supersedes their liveness/tool-calling use case with a
persisted, cross-provider, scored equivalent; they're still the only game in
town for the embedding-specific and vision-specific checks they do that this
service doesn't cover. Not retired automatically — flag to the owner if you
want them archived now that this exists.
