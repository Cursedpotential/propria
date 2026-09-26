# Cross-provider plain-chat liveness — 2026-08-29

> _Byline: Codex · streamed provider probe + headless OpenCode CLI_

- Scope: every candidate discovered from configured direct provider catalogs, OpenRouter free models only, plus all seven OpenCode-hosted models. Kimi for Coding was the only explicitly skipped provider.
- Method: exact `PONG`; SSE streams consumed through completion; two initial 20-second misses were rechecked with 60 seconds and both passed.

## Provider summary

| Provider | Tested | Visible text | Other / failed |
|---|---:|---:|---:|
| cerebras | 2 | 0 | 2 |
| groq | 12 | 7 | 5 |
| mistral | 46 | 35 | 11 |
| ollama_cloud | 19 | 19 | 0 |
| openai | 78 | 20 | 58 |
| opencode | 7 | 7 | 0 |
| openrouter_free | 18 | 14 | 4 |

## Model-by-model results

| Provider | Model | Result | Exact PONG | First event | Total | HTTP |
|---|---|---|---|---:|---:|---:|
| cerebras | `gemma-4-31b` | http_error | no | - | 0.18 | 402 |
| cerebras | `gpt-oss-120b` | http_error | no | - | 0.4 | 402 |
| groq | `allam-2-7b` | visible_text | no | 0.39 | 0.39 | 200 |
| groq | `canopylabs/orpheus-arabic-saudi` | http_error | no | - | 0.41 | 400 |
| groq | `canopylabs/orpheus-v1-english` | http_error | no | - | 0.65 | 400 |
| groq | `groq/compound` | reasoning_only | no | 1.18 | 1.27 | 200 |
| groq | `groq/compound-mini` | visible_text | yes | 0.43 | 0.75 | 200 |
| groq | `meta-llama/llama-prompt-guard-2-22m` | http_error | no | - | 0.44 | 400 |
| groq | `meta-llama/llama-prompt-guard-2-86m` | http_error | no | - | 0.58 | 400 |
| groq | `openai/gpt-oss-120b` | visible_text | yes | 0.62 | 0.72 | 200 |
| groq | `openai/gpt-oss-20b` | visible_text | yes | 0.32 | 0.37 | 200 |
| groq | `openai/gpt-oss-safeguard-20b` | visible_text | yes | 0.7 | 0.75 | 200 |
| groq | `qwen/qwen3.6-27b` | visible_text | no | 0.93 | 1.12 | 200 |
| groq | `qwen/qwen3.8-27b` | visible_text | yes | 0.64 | 0.64 | 200 |
| mistral | `codestral-2508` | visible_text | yes | 0.44 | 0.44 | 200 |
| mistral | `codestral-latest` | visible_text | yes | 0.59 | 0.6 | 200 |
| mistral | `devstral-2512` | visible_text | yes | 0.66 | 0.74 | 200 |
| mistral | `devstral-latest` | visible_text | yes | 0.79 | 0.83 | 200 |
| mistral | `devstral-medium-latest` | visible_text | yes | 8.35 | 8.4 | 200 |
| mistral | `glm-5-2` | visible_text | yes | 1.22 | 1.22 | 200 |
| mistral | `labs-leanstral-1-5` | http_error | no | - | 0.32 | 403 |
| mistral | `labs-leanstral-1-5-1` | http_error | no | - | 0.2 | 403 |
| mistral | `magistral-medium-latest` | visible_text | yes | 0.4 | 0.42 | 200 |
| mistral | `magistral-small-latest` | visible_text | yes | 1.61 | 1.62 | 200 |
| mistral | `ministral-14b-2512` | visible_text | yes | 0.44 | 0.44 | 200 |
| mistral | `ministral-14b-latest` | visible_text | yes | 0.55 | 0.55 | 200 |
| mistral | `ministral-3b-2512` | visible_text | yes | 0.5 | 0.52 | 200 |
| mistral | `ministral-3b-latest` | visible_text | yes | 0.47 | 0.47 | 200 |
| mistral | `ministral-8b-2512` | visible_text | yes | 0.47 | 0.48 | 200 |
| mistral | `ministral-8b-latest` | visible_text | yes | 0.43 | 0.45 | 200 |
| mistral | `mistral-code-agent-latest` | visible_text | yes | 0.47 | 0.5 | 200 |
| mistral | `mistral-code-fim-latest` | visible_text | yes | 0.41 | 0.42 | 200 |
| mistral | `mistral-code-latest` | visible_text | yes | 0.42 | 0.43 | 200 |
| mistral | `mistral-large-2512` | visible_text | yes | 1.23 | 1.26 | 200 |
| mistral | `mistral-large-latest` | visible_text | yes | 0.57 | 0.64 | 200 |
| mistral | `mistral-medium` | visible_text | yes | 0.46 | 0.49 | 200 |
| mistral | `mistral-medium-2505` | visible_text | yes | 0.5 | 0.53 | 200 |
| mistral | `mistral-medium-2508` | visible_text | yes | 0.48 | 0.48 | 200 |
| mistral | `mistral-medium-2604` | visible_text | yes | 0.5 | 0.52 | 200 |
| mistral | `mistral-medium-3` | visible_text | yes | 0.53 | 0.53 | 200 |
| mistral | `mistral-medium-3-5` | visible_text | yes | 0.47 | 0.49 | 200 |
| mistral | `mistral-medium-3.5` | visible_text | yes | 0.59 | 0.59 | 200 |
| mistral | `mistral-medium-latest` | visible_text | yes | 0.54 | 0.57 | 200 |
| mistral | `mistral-ocr-2512` | http_error | no | - | 0.34 | 400 |
| mistral | `mistral-ocr-3` | http_error | no | - | 0.23 | 400 |
| mistral | `mistral-ocr-3-0` | http_error | no | - | 0.25 | 400 |
| mistral | `mistral-ocr-4` | http_error | no | - | 0.29 | 400 |
| mistral | `mistral-ocr-4-0` | http_error | no | - | 0.24 | 400 |
| mistral | `mistral-ocr-4-1` | http_error | no | - | 0.22 | 400 |
| mistral | `mistral-ocr-latest` | http_error | no | - | 0.34 | 400 |
| mistral | `mistral-small-2603` | visible_text | yes | 0.53 | 0.53 | 200 |
| mistral | `mistral-small-latest` | visible_text | yes | 0.57 | 0.58 | 200 |
| mistral | `mistral-vibe-cli-fast` | visible_text | yes | 0.55 | 0.55 | 200 |
| mistral | `mistral-vibe-cli-latest` | visible_text | yes | 0.47 | 0.49 | 200 |
| mistral | `mistral-vibe-cli-with-tools` | visible_text | yes | 0.6 | 0.62 | 200 |
| mistral | `voxtral-mini-2602` | http_error | no | - | 0.35 | 400 |
| mistral | `voxtral-mini-latest` | http_error | no | - | 0.26 | 400 |
| mistral | `voxtral-small-2507` | visible_text | yes | 0.45 | 0.45 | 200 |
| mistral | `voxtral-small-latest` | visible_text | yes | 0.53 | 0.53 | 200 |
| mistral | `zai-glm-5-2` | visible_text | yes | 0.58 | 0.6 | 200 |
| ollama_cloud | `deepseek-v4-flash:0731` | visible_text | yes | 0.46 | 0.57 | 200 |
| ollama_cloud | `deepseek-v4-pro:0813` | visible_text | yes | 0.87 | 0.96 | 200 |
| ollama_cloud | `gemma4:31b` | visible_text | yes | 0.88 | 0.99 | 200 |
| ollama_cloud | `glm-5.1` | visible_text | yes | 1.37 | 2.41 | 200 |
| ollama_cloud | `glm-5.2` | visible_text | yes | 1.34 | 2.05 | 200 |
| ollama_cloud | `glm-5.3` | visible_text | yes | 0.74 | 1.1 | 200 |
| ollama_cloud | `glm-5.3-flash` | visible_text | yes | 0.82 | 1.26 | 200 |
| ollama_cloud | `gpt-oss:120b` | visible_text | yes | 0.57 | 0.98 | 200 |
| ollama_cloud | `gpt-oss:20b` | visible_text | yes | 0.84 | 1.47 | 200 |
| ollama_cloud | `kimi-k2.6` | visible_text | yes | 0.96 | 1.81 | 200 |
| ollama_cloud | `kimi-k2.7-code` | visible_text | yes | 0.81 | 1.01 | 200 |
| ollama_cloud | `kimi-k3` | visible_text | yes | 0.92 | 1.5 | 200 |
| ollama_cloud | `minimax-m2.7` | visible_text | yes | 1.14 | 2.54 | 200 |
| ollama_cloud | `minimax-m3` | visible_text | yes | 0.64 | 0.82 | 200 |
| ollama_cloud | `mistral-large-3:675b` | visible_text | yes | 0.76 | 0.86 | 200 |
| ollama_cloud | `nemotron-3-nano:30b` | visible_text | yes | 0.61 | 1.12 | 200 |
| ollama_cloud | `nemotron-3-super` | visible_text | yes | 0.57 | 0.79 | 200 |
| ollama_cloud | `nemotron-3-ultra` | visible_text | yes | 0.72 | 0.98 | 200 |
| ollama_cloud | `qwen3.5:397b` | visible_text | yes | 1.14 | 4.57 | 200 |
| openai | `babbage-002` | http_error | no | - | 1.8 | 404 |
| openai | `chat-latest` | http_error | no | - | 1.79 | 400 |
| openai | `davinci-002` | http_error | no | - | 0.57 | 404 |
| openai | `gpt-3.5-turbo` | visible_text | yes | 2.01 | 2.05 | 200 |
| openai | `gpt-3.5-turbo-0125` | visible_text | yes | 2.33 | 2.4 | 200 |
| openai | `gpt-3.5-turbo-1106` | visible_text | yes | 2.09 | 2.13 | 200 |
| openai | `gpt-3.5-turbo-16k` | visible_text | yes | 2.14 | 2.17 | 200 |
| openai | `gpt-3.5-turbo-instruct` | http_error | no | - | 0.68 | 404 |
| openai | `gpt-3.5-turbo-instruct-0914` | http_error | no | - | 0.1 | 404 |
| openai | `gpt-4` | visible_text | yes | 1.46 | 1.51 | 200 |
| openai | `gpt-4-0613` | visible_text | yes | 0.49 | 0.54 | 200 |
| openai | `gpt-4-turbo` | visible_text | yes | 0.61 | 0.67 | 200 |
| openai | `gpt-4-turbo-2024-04-09` | visible_text | yes | 1.85 | 1.92 | 200 |
| openai | `gpt-4.1` | visible_text | yes | 0.49 | 0.54 | 200 |
| openai | `gpt-4.1-2025-04-14` | visible_text | yes | 0.47 | 0.5 | 200 |
| openai | `gpt-4.1-mini` | visible_text | yes | 0.51 | 0.56 | 200 |
| openai | `gpt-4.1-mini-2025-04-14` | visible_text | yes | 0.52 | 0.56 | 200 |
| openai | `gpt-4.1-nano` | visible_text | yes | 0.52 | 0.59 | 200 |
| openai | `gpt-4.1-nano-2025-04-14` | visible_text | yes | 0.5 | 0.58 | 200 |
| openai | `gpt-4o` | visible_text | yes | 0.51 | 0.56 | 200 |
| openai | `gpt-4o-2024-05-13` | visible_text | yes | 0.46 | 0.51 | 200 |
| openai | `gpt-4o-2024-08-06` | visible_text | yes | 0.43 | 0.48 | 200 |
| openai | `gpt-4o-2024-11-20` | visible_text | yes | 0.46 | 0.5 | 200 |
| openai | `gpt-4o-mini` | visible_text | yes | 0.41 | 0.46 | 200 |
| openai | `gpt-4o-mini-2024-07-18` | visible_text | yes | 0.48 | 0.56 | 200 |
| openai | `gpt-5` | http_error | no | - | 0.21 | 400 |
| openai | `gpt-5-2025-08-07` | http_error | no | - | 0.1 | 400 |
| openai | `gpt-5-chat-latest` | http_error | no | - | 0.08 | 404 |
| openai | `gpt-5-codex` | http_error | no | - | 0.08 | 404 |
| openai | `gpt-5-mini` | http_error | no | - | 0.23 | 400 |
| openai | `gpt-5-mini-2025-08-07` | http_error | no | - | 0.11 | 400 |
| openai | `gpt-5-nano` | http_error | no | - | 0.17 | 400 |
| openai | `gpt-5-nano-2025-08-07` | http_error | no | - | 0.11 | 400 |
| openai | `gpt-5-pro` | http_error | no | - | 0.11 | 404 |
| openai | `gpt-5-pro-2025-10-06` | http_error | no | - | 0.16 | 404 |
| openai | `gpt-5-search-api` | http_error | no | - | 0.13 | 400 |
| openai | `gpt-5-search-api-2025-10-14` | http_error | no | - | 0.11 | 400 |
| openai | `gpt-5.1` | http_error | no | - | 0.1 | 400 |
| openai | `gpt-5.1-2025-11-13` | http_error | no | - | 0.1 | 400 |
| openai | `gpt-5.1-chat-latest` | http_error | no | - | 0.09 | 404 |
| openai | `gpt-5.1-codex` | http_error | no | - | 0.09 | 404 |
| openai | `gpt-5.1-codex-max` | http_error | no | - | 0.59 | 404 |
| openai | `gpt-5.1-codex-mini` | http_error | no | - | 0.09 | 404 |
| openai | `gpt-5.2` | http_error | no | - | 0.48 | 400 |
| openai | `gpt-5.2-2025-12-11` | http_error | no | - | 0.1 | 400 |
| openai | `gpt-5.2-chat-latest` | http_error | no | - | 0.08 | 404 |
| openai | `gpt-5.2-codex` | http_error | no | - | 0.08 | 404 |
| openai | `gpt-5.2-pro` | http_error | no | - | 0.12 | 404 |
| openai | `gpt-5.2-pro-2025-12-11` | http_error | no | - | 0.12 | 404 |
| openai | `gpt-5.3-chat-latest` | http_error | no | - | 0.1 | 404 |
| openai | `gpt-5.3-codex` | http_error | no | - | 0.08 | 404 |
| openai | `gpt-5.4` | http_error | no | - | 0.12 | 400 |
| openai | `gpt-5.4-2026-03-05` | http_error | no | - | 0.1 | 400 |
| openai | `gpt-5.4-mini` | http_error | no | - | 0.14 | 400 |
| openai | `gpt-5.4-mini-2026-03-17` | http_error | no | - | 0.12 | 400 |
| openai | `gpt-5.4-nano` | http_error | no | - | 0.1 | 400 |
| openai | `gpt-5.4-nano-2026-03-17` | http_error | no | - | 0.12 | 400 |
| openai | `gpt-5.4-pro` | http_error | no | - | 0.11 | 404 |
| openai | `gpt-5.4-pro-2026-03-05` | http_error | no | - | 0.11 | 404 |
| openai | `gpt-5.5` | http_error | no | - | 0.12 | 400 |
| openai | `gpt-5.5-2026-04-23` | http_error | no | - | 0.11 | 400 |
| openai | `gpt-5.5-pro` | http_error | no | - | 0.12 | 404 |
| openai | `gpt-5.5-pro-2026-04-23` | http_error | no | - | 0.22 | 404 |
| openai | `gpt-5.6-luna` | http_error | no | - | 0.12 | 400 |
| openai | `gpt-5.6-sol` | http_error | no | - | 0.12 | 400 |
| openai | `gpt-5.6-terra` | http_error | no | - | 0.13 | 400 |
| openai | `o1` | http_error | no | - | 0.14 | 400 |
| openai | `o1-2024-12-17` | http_error | no | - | 0.12 | 400 |
| openai | `o1-pro` | http_error | no | - | 0.12 | 404 |
| openai | `o1-pro-2025-03-19` | http_error | no | - | 0.11 | 404 |
| openai | `o3` | http_error | no | - | 0.12 | 400 |
| openai | `o3-2025-04-16` | http_error | no | - | 0.12 | 400 |
| openai | `o3-mini` | http_error | no | - | 0.13 | 400 |
| openai | `o3-mini-2025-01-31` | http_error | no | - | 0.16 | 400 |
| openai | `o4-mini` | http_error | no | - | 0.13 | 400 |
| openai | `o4-mini-2025-04-16` | http_error | no | - | 0.11 | 400 |
| openai | `sora-2` | http_error | no | - | 0.11 | 503 |
| openai | `sora-2-pro` | http_error | no | - | 0.09 | 503 |
| opencode | `big-pickle` | visible_text | yes | - | - | 200 |
| opencode | `hy3-free` | visible_text | yes | - | - | 200 |
| opencode | `ling-3.0-flash-fin-free` | visible_text | yes | - | - | 200 |
| opencode | `mimo-v2.5-free` | visible_text | yes | - | - | 200 |
| opencode | `muse-spark-1.2-contributor-free` | visible_text | yes | - | - | 200 |
| opencode | `nemotron-3-ultra-free` | visible_text | yes | - | - | 200 |
| opencode | `nemotron-3.5-lightning-free` | visible_text | yes | - | - | 200 |
| openrouter_free | `cohere/north-mini-code:free` | visible_text | yes | 0.36 | 0.54 | 200 |
| openrouter_free | `dots-studio/dots-3-note-preview:free` | visible_text | yes | 0.79 | 1.26 | 200 |
| openrouter_free | `google/gemma-4-26b-a4b-it:free` | visible_text | yes | 0.93 | 0.95 | 200 |
| openrouter_free | `google/gemma-4-31b-it:free` | visible_text | yes | 1.21 | 1.22 | 200 |
| openrouter_free | `inclusionai/ling-3.0-flash-fin:free` | visible_text | yes | 1.15 | 2.61 | 200 |
| openrouter_free | `liquid/lfm-2.5-2.6b:free` | visible_text | yes | 1.11 | 1.26 | 200 |
| openrouter_free | `minimax/minimax-m2.7:free` | visible_text | yes | 1.79 | 4.05 | 200 |
| openrouter_free | `minimax/minimax-m3:free` | visible_text | yes | 1.66 | 1.8 | 200 |
| openrouter_free | `nvidia/nemotron-3-nano-omni-30b-a3b-reasoning:free` | visible_text | yes | 0.82 | 1.69 | 200 |
| openrouter_free | `nvidia/nemotron-3-super-120b-a12b:free` | visible_text | yes | 0.74 | 1.01 | 200 |
| openrouter_free | `nvidia/nemotron-3-ultra-550b-a55b:free` | visible_text | yes | 40.86 | 40.99 | 200 |
| openrouter_free | `nvidia/nemotron-3.5-content-safety:free` | visible_text | no | 0.36 | 2.0 | 200 |
| openrouter_free | `nvidia/nemotron-3.5-lightning:free` | visible_text | yes | 5.78 | 8.62 | 200 |
| openrouter_free | `poolside/laguna-s-2.1:free` | visible_text | yes | 0.88 | 0.92 | 200 |
| openrouter_free | `poolside/laguna-xs-2.1:free` | http_error | no | - | 0.26 | 429 |
| openrouter_free | `thinkingmachines/inkling-small:free` | http_error | no | - | 0.06 | 403 |
| openrouter_free | `thinkingmachines/inkling:free` | http_error | no | - | 0.08 | 403 |
| openrouter_free | `z-ai/glm-5.2:free` | http_error | no | - | 0.28 | 429 |

## Discovery boundary

- OpenCode CLI exposed 439 IDs: Kimi for Coding 4, Ollama Cloud 21, OpenAI 52, OpenCode 7, OpenRouter 355.
- The direct provider APIs exposed additional Groq, Mistral, and Cerebras catalogs already configured in the secrets store; those were also tested.
- HTTP errors remain visible in the raw JSON with provider response details.
