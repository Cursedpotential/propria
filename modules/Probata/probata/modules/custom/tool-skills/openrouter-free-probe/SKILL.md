---
name: openrouter-free-probe
description: Live-test every OpenRouter FREE model (":free" ids / zero-priced) for plain chat, tool calling, and vision (image input), recording latency, prompt/completion token usage, and OpenRouter's own capability claims (input_modalities, supported_parameters, context) next to the live result. Saves every run to runs/ with a diff vs the previous run; any single check can run alone via --tests chat|tools|vision. Use when asked "which OpenRouter free models work", "free models with tool calling / vision", "is <model>:free still up / rate-limited", or before routing a Portkey fallback chain, n8n workflow, or agent to a zero-cost model.
---

# openrouter-free-probe

> _Byline: Claude Code · Fable 5 · 2026-08-26_

Third of the probe family (`nim-embed-probe`, `nim-chat-probe`). Key = `OPENROUTER_API_KEY` env var (never paste it).

```
python ~/.claude/skills/openrouter-free-probe/scripts/openrouter_free_probe.py                 # all free models, 3 checks
python ~/.claude/skills/openrouter-free-probe/scripts/openrouter_free_probe.py --tests tools   # ONE check (chat | tools | vision)
python ~/.claude/skills/openrouter-free-probe/scripts/openrouter_free_probe.py --only org/x:free
python ~/.claude/skills/openrouter-free-probe/scripts/openrouter_free_probe.py --extra org/paid-model   # non-free ids too
python ~/.claude/skills/openrouter-free-probe/scripts/openrouter_free_probe.py --workers 3     # default 2 (free tier ≈ 20 req/min; 429s auto-backoff ×3)
python ~/.claude/skills/openrouter-free-probe/scripts/openrouter_free_probe.py --last | --history
```

**Recall first** — `--last` before re-probing (free-tier calls count against the daily cap). Partial runs (`--only`/`--tests`) overlay onto the previous run so `latest.md` is always the full picture.

**Columns unique to this tool**: `claims tools` / `claims image` = what `/models` advertises vs `tool call` / `vision` = what actually happened. A `claims yes / live no` row is provider drift; a `claims no / live yes` row is a bonus.

**Reading failures**: `429` after 3 backoffs = free-tier rate/daily limit (retry later, not a dead model); `404` = model delisted; `200` with empty `choices` = upstream provider error (recorded as FAIL with the error text).

Mirror copy: `E:\AI_Workspace\Projects\the-platform-workspace\tool-skills\openrouter-free-probe\`.
