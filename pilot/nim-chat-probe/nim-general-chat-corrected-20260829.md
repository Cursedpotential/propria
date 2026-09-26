# NVIDIA NIM general-chat liveness — corrected streamed report

> _Byline: Codex · streamed plain-chat probe · 2026-08-29_

- Catalog: 83 models; general-chat candidates: 64; specialized endpoints excluded: 19
- Method: stream `POST /v1/chat/completions`; wait up to 20 seconds for first SSE event, then recheck that slow set with 60 seconds. Once streaming begins, consume through `[DONE]` without a whole-answer timeout.
- Good general chat: 13 models with visible text; 13 followed the exact PONG instruction.
- Unavailable: 46 catalog entries returned 404/410. Non-text stream: 2. No headers after 60 seconds: 3.

## Good general-chat models

| Model | First streamed text | Full response | Exact PONG |
|---|---:|---:|---|
| `meta/llama-3.2-11b-vision-instruct` | 0.2s | 0.24s | yes |
| `nvidia/nemotron-3-nano-30b-a3b` | 0.29s | 0.76s | yes |
| `minimaxai/minimax-m3` | 0.31s | 0.37s | yes |
| `openai/gpt-oss-20b` | 0.37s | 0.72s | yes |
| `nvidia/nemotron-3.5-lightning-30b-a3b` | 0.43s | 3.08s | yes |
| `nvidia/nemotron-3-nano-omni-30b-a3b-reasoning` | 0.44s | 2.16s | yes |
| `nvidia/nemotron-3-super-120b-a12b` | 0.86s | 2.04s | yes |
| `moonshotai/kimi-k3` | 0.87s | 1.57s | yes |
| `openai/gpt-oss-120b` | 0.87s | 1.27s | yes |
| `mistralai/mistral-nemotron` | 0.88s | 0.94s | yes |
| `meta/llama-3.2-90b-vision-instruct` | 1.29s | 1.89s | yes |
| `meta/muse-glimmer-30b` | 2.3s | 14.78s | yes |
| `nvidia/nemotron-3-ultra-550b-a55b` | 34.85s | 35.48s | yes |

## Not usable as general chat in this run

| Model | Result | Status |
|---|---|---:|
| `01-ai/yi-large` | http_error | 404 |
| `adept/fuyu-8b` | http_error | 404 |
| `ai21labs/jamba-1.5-large-instruct` | http_error | 404 |
| `aisingapore/sea-lion-7b-instruct` | http_error | 404 |
| `bigcode/starcoder2-15b` | http_error | 404 |
| `databricks/dbrx-instruct` | http_error | 404 |
| `deepseek-ai/deepseek-coder-6.7b-instruct` | http_error | 404 |
| `deepseek-ai/deepseek-v4-flash-0731` | no_headers | - |
| `deepseek-ai/deepseek-v4-pro-0813` | no_headers | - |
| `google/codegemma-1.1-7b` | http_error | 404 |
| `google/codegemma-7b` | http_error | 404 |
| `google/deplot` | http_error | 404 |
| `google/diffusiongemma-26b-a4b-it` | events_without_text | 200 |
| `google/gemma-2b` | http_error | 404 |
| `google/gemma-3-12b-it` | http_error | 404 |
| `google/gemma-3-4b-it` | http_error | 404 |
| `google/gemma-4-31b-it` | no_headers | - |
| `google/recurrentgemma-2b` | http_error | 404 |
| `ibm/granite-3.0-3b-a800m-instruct` | http_error | 404 |
| `ibm/granite-3.0-8b-instruct` | http_error | 404 |
| `ibm/granite-34b-code-instruct` | http_error | 404 |
| `ibm/granite-8b-code-instruct` | http_error | 404 |
| `meta/codellama-70b` | http_error | 404 |
| `meta/llama2-70b` | http_error | 404 |
| `microsoft/kosmos-2` | http_error | 404 |
| `microsoft/phi-3-vision-128k-instruct` | http_error | 404 |
| `microsoft/phi-3.5-moe-instruct` | http_error | 404 |
| `mistralai/codestral-22b-instruct-v0.1` | http_error | 404 |
| `mistralai/mistral-7b-instruct-v0.3` | http_error | 404 |
| `mistralai/mistral-large` | http_error | 404 |
| `mistralai/mistral-large-2-instruct` | http_error | 404 |
| `mistralai/mixtral-8x22b-v0.1` | http_error | 404 |
| `moonshotai/kimi-k2.6` | http_error | 404 |
| `nv-mistralai/mistral-nemo-12b-instruct` | http_error | 404 |
| `nvidia/cosmos-reason2-8b` | http_error | 404 |
| `nvidia/llama-3.1-nemotron-51b-instruct` | http_error | 404 |
| `nvidia/llama-3.1-nemotron-70b-instruct` | http_error | 404 |
| `nvidia/llama-3.1-nemotron-ultra-253b-v1` | http_error | 404 |
| `nvidia/llama3-chatqa-1.5-70b` | http_error | 404 |
| `nvidia/mistral-nemo-minitron-8b-8k-instruct` | http_error | 404 |
| `nvidia/nemotron-4-340b-instruct` | http_error | 404 |
| `nvidia/nemotron-4-340b-reward` | http_error | 404 |
| `nvidia/nemotron-nano-3-30b-a3b` | http_error | 404 |
| `nvidia/neva-22b` | http_error | 404 |
| `nvidia/vila` | http_error | 404 |
| `poolside/laguna-xs-2.1` | events_without_text | 200 |
| `writer/palmyra-creative-122b` | http_error | 404 |
| `writer/palmyra-fin-70b-32k` | http_error | 404 |
| `writer/palmyra-med-70b` | http_error | 404 |
| `writer/palmyra-med-70b-32k` | http_error | 404 |
| `zyphra/zamba2-7b-instruct` | http_error | 404 |

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
