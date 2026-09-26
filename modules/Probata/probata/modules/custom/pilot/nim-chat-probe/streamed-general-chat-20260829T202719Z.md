# NVIDIA NIM general-chat liveness — 2026-08-29T20:27:19+00:00

> _Byline: Codex · streamed plain-chat probe · 2026-08-29_

- Catalog: 83 models; general-chat candidates: 7; excluded specialized endpoints: 19
- Method: SSE streaming; 60.0s maximum wait for response/first event; once streaming starts, read continues until `[DONE]` with no whole-answer timeout.
- Visible chat response: 2; exact `PONG`: 2; unavailable (404/410): 0; unresolved: 5

## General chat models that answered

| Model | First token | Total | Exact PONG | Finish |
|---|---:|---:|---|---|
| `moonshotai/kimi-k3` | 0.87s | 1.57s | yes | stop |
| `nvidia/nemotron-3-ultra-550b-a55b` | 34.85s | 35.48s | yes | stop |

## Unavailable or unresolved

| Model | Outcome | Status | Detail |
|---|---|---:|---|
| `deepseek-ai/deepseek-v4-flash-0731` | no_headers | - |  |
| `deepseek-ai/deepseek-v4-pro-0813` | no_headers | - |  |
| `google/diffusiongemma-26b-a4b-it` | events_without_text | 200 |  |
| `google/gemma-4-31b-it` | no_headers | - |  |
| `poolside/laguna-xs-2.1` | events_without_text | 200 |  |

## Excluded specialized endpoints

- `meta/llama-guard-4-12b`
- `nvidia/ai-synthetic-video-detector`
- `nvidia/embed-qa-4`
- `nvidia/ising-calibration-1.5-31b`
- `nvidia/llama-3.1-nemoguard-8b-content-safety`
- `nvidia/llama-3.1-nemoguard-8b-topic-control`
- `nvidia/llama-3.1-nemotron-safety-guard-8b-v3`
- `nvidia/llama-3.2-nemoretriever-1b-vlm-embed-v1`
- `nvidia/llama-3.2-nv-embedqa-1b-v1`
- `nvidia/llama-nemotron-embed-vl-1b-v2`
- `nvidia/nemotron-3-embed-1b`
- `nvidia/nemotron-3.5-content-safety`
- `nvidia/nemotron-parse`
- `nvidia/nv-embedqa-mistral-7b-v2`
- `nvidia/nvclip`
- `nvidia/riva-translate-4b-instruct`
- `nvidia/riva-translate-4b-instruct-v1.1`
- `nvidia/riva-translate-4b-instruct-v2`
- `snowflake/arctic-embed-l`
