---
name: nim-chat-probe
description: Live-test every NVIDIA NIM chat/text model (integrate.api.nvidia.com) for plain chat, tool calling, and vision (image input), recording latency + prompt/completion token usage per call. Lists /models, skips embedders, runs the checks concurrently, and saves every run to runs/ with a diff vs the previous run. Any single check can be run alone via --tests chat|tools|vision. Use when asked "which NIM chat models work", "which NIM models support tool calling / function calling", "which NIM models are vision-capable / take images", "how fast is <model> on NIM", "is <model> retired", or before picking a chat model for an agent, Portkey route, or n8n workflow.
---

# nim-chat-probe

> _Byline: Claude Code · Fable 5 · 2026-08-26_

Sister tool of `nim-embed-probe` (embedders). Key = `NVIDIA_API_KEY` (User-scope registry env var; never paste it).

```
python ~/.claude/skills/nim-chat-probe/scripts/nim_chat_probe.py                    # all models, all 3 checks, save + diff
python ~/.claude/skills/nim-chat-probe/scripts/nim_chat_probe.py --tests vision     # ONE check only (chat | tools | vision, comma-list ok)
python ~/.claude/skills/nim-chat-probe/scripts/nim_chat_probe.py --only org/model   # specific model(s)
python ~/.claude/skills/nim-chat-probe/scripts/nim_chat_probe.py --extra org/model  # ids not in /models (retired?)
python ~/.claude/skills/nim-chat-probe/scripts/nim_chat_probe.py --workers 8        # concurrency (default 6)
python ~/.claude/skills/nim-chat-probe/scripts/nim_chat_probe.py --last             # recall newest report, no API calls
python ~/.claude/skills/nim-chat-probe/scripts/nim_chat_probe.py --history          # one line per saved run
```

**Recall first** — `--last` before re-probing; `runs/*.json` hold every past run (UTC-stamped).

**What each check does** (all real `/chat/completions` calls, `temperature 0`):
- `chat`  — "Reply with exactly the word PONG." → latency, `usage.prompt_tokens/completion_tokens`, flags `R` if the model returns a `reasoning_content` field (reasoning models cost 20× per memory `nim-chat-model-benchmark`).
- `tools` — one `get_weather` function + "What's the weather in Paris? Use the tool." → `CALL` only if `tool_calls[]` came back; `text` = answered in prose (no real tool support); `ERR 4xx` = rejected the `tools` param.
- `vision` — 1×1 PNG as `image_url` data-URI + "What color is this image?" → `OK` = accepted multimodal content parts; `no 4xx` = text-only model.
- If `chat` fails the other two are skipped for that model (saves calls). With `--tests` excluding chat, every listed model is hit with just the chosen check.

**Reading failures**: `410 … end of life on <date>` = retired (date authoritative); `404 Function … Not found for account` = listed but not entitled; `400` on tools/vision = feature unsupported, model itself fine.

Mirror copy: `E:\AI_Workspace\Projects\the-platform-workspace\tool-skills\nim-chat-probe\`.
