---
name: nim-embed-probe
description: Live-test which NVIDIA NIM embedding models actually work right now (integrate.api.nvidia.com). Lists /models, filters embedders, POSTs a real 4-text batch to each (retrying with input_type=passage for asymmetric models), always re-probes known/retired ids so EOLs are caught, and saves every run to runs/ with a diff vs the previous run. Use when asked "what embedders are available on NIM", "is <model> still up", "did NIM retire X", before picking/changing an embed model, or when an embed pipeline suddenly returns 404/410/empty vectors.
---

# nim-embed-probe

> _Byline: Claude Code · Fable 5 · 2026-08-26_

**Run** (key = `NVIDIA_API_KEY`, User-scope registry env var on this box; never paste it):

```
python ~/.claude/skills/nim-embed-probe/scripts/nim_embed_probe.py            # probe + save + diff
python ~/.claude/skills/nim-embed-probe/scripts/nim_embed_probe.py --last     # recall latest report (no API calls)
python ~/.claude/skills/nim-embed-probe/scripts/nim_embed_probe.py --history  # one line per saved run
python ~/.claude/skills/nim-embed-probe/scripts/nim_embed_probe.py --extra org/model-id   # probe extra ids
```

**Recall first** — before re-hitting the API, `--last` shows the newest saved result; `runs/*.json` are the raw per-run records (timestamped UTC).

**Reading the output**
- `OK … symmetric` → works with a plain OpenAI-style `/embeddings` call.
- `OK … asymmetric(input_type required)` → works only with `input_type: passage|query` in the body (memory: `nim-embedqa-scoped-not-banned` — a per-client constraint, not a ban).
- `410 … end of life on <date>` → NVIDIA retired it; the date is authoritative. Grep configs for the id and repoint.
- `404 Function … Not found for account` → model is listed but not entitled/served for this account.
- `UNLISTED` → id was probed from the KNOWN list or `--extra`, not from `/models`.

**Known result 2026-08-26**: only `nvidia/nemotron-3-embed-1b` (2048-d, symmetric) and `nvidia/llama-nemotron-embed-vl-1b-v2` (2048-d, needs input_type) work. `nv-embed-v1` + `bge-m3` EOL'd 2026-08-25.

Mirror copy: `E:\AI_Workspace\Projects\the-platform-workspace\tool-skills\nim-embed-probe\`.
