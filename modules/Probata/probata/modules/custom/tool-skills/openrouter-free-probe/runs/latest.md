# OpenRouter FREE models — live probe 2026-08-26T12:23:54+00:00

> _Byline: openrouter-free-probe · 2026-08-26_

- `/models` listed 417 models, 21 free; 21 probed; 16 alive; 13 emitted a tool call; 8 accepted an image
- endpoint: `https://openrouter.ai/api/v1`

## WORKING (sorted by chat latency)

| model | ctx | chat s | prompt/completion tok | reasoning | tool call | claims tools | tools s | vision | claims image | vision s | vision reply |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `nvidia/nemotron-3.5-content-safety:free` | 128000 | 0.48 | 470/16 | yes | ERR 404 | no | 0.19 | yes | yes | 0.79 |  |
| `cohere/north-mini-code:free` | 256000 | 0.49 | 8/16 | yes | CALL get_weather | yes | 0.78 | no (404) | no | 0.59 |  |
| `nvidia/nemotron-3-super-120b-a12b:free` | 262144 | 0.56 | 24/16 | yes | CALL get_weather | yes | 1.04 | no (404) | no | 0.19 |  |
| `poolside/laguna-xs-2.1:free` | 262144 | 0.65 | 51/16 | yes | ERR 429 | yes | 25.56 | no (404) | no | 0.2 |  |
| `nvidia/nemotron-3.5-lightning:free` | 1000000 | 0.71 | 24/16 | yes | CALL get_weather | yes | 1.29 | no (404) | no | 0.21 |  |
| `nvidia/nemotron-3-nano-omni-30b-a3b-reasoning:free` | 256000 | 0.85 | 24/61 | yes | CALL get_weather | yes | 2.45 | yes | yes | 7.99 | Red |
| `liquid/lfm-2.5-2.6b:free` | 65536 | 0.88 | 18/16 | yes | CALL get_weather | yes | 1.04 | no (404) | no | 0.23 |  |
| `openrouter/free` | 200000 | 1.06 | 9/2 |  | CALL get_weather | yes | 1.49 | yes | yes | 1.77 | Red |
| `google/gemma-4-31b-it:free` | 262144 | 1.12 | 9/2 |  | CALL get_weather | yes | 1.52 | yes | yes | 1.01 | Red |
| `google/gemma-4-26b-a4b-it:free` | 262144 | 1.16 | 9/2 |  | CALL get_weather | yes | 1.36 | yes | yes | 1.94 | Red |
| `dots-studio/dots-3-note-preview:free` | 512000 | 1.23 | 19/16 | yes | CALL get_weather | yes | 1.92 | yes | yes | 1.47 |  |
| `minimax/minimax-m2.7:free` | 196608 | 1.72 | 49/16 | yes | CALL get_weather | yes | 3.97 | no (404) | no | 0.24 |  |
| `nvidia/nemotron-3-ultra-550b-a55b:free` | 1000000 | 2.86 | 24/16 | yes | ERR 200 | yes | 0.44 | no (404) | no | 0.18 |  |
| `stealth/ox-alpha` | 1048576 | 6.43 | 95/16 | yes | CALL get_weather | yes | 6.36 | yes | yes | 34.87 | Red |
| `poolside/laguna-s-2.1:free` | 262144 | 13.38 | 51/3 |  | CALL get_weather | yes | 14.79 | no (404) | no | 0.17 |  |
| `minimax/minimax-m3:free` | 1048576 | 19.75 | 171/2 |  | CALL get_weather | yes | 1.86 | yes | yes | 10.42 | Black |

## NOT WORKING

| model | status | detail |
|---|---|---|
| `google/lyria-3-clip-preview` | 402 | Insufficient credits. Add more using https://openrouter.ai/settings/credits |
| `google/lyria-3-pro-preview` | 402 | Insufficient credits. Add more using https://openrouter.ai/settings/credits |
| `thinkingmachines/inkling-small:free` | 403 | thinkingmachines/inkling-small:free is only available on agentic harnesses. Try plugging it into a coding agent or productivity app listed on https://openrouter |
| `thinkingmachines/inkling:free` | 403 | thinkingmachines/inkling:free is only available on agentic harnesses. Try plugging it into a coding agent or productivity app listed on https://openrouter.ai/ap |
| `z-ai/glm-5.2:free` | 429 | Provider returned error |

## Changes vs previous run

- (no previous run / no changes)
