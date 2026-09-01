# NIM chat/text models — live probe 2026-08-26T12:21:15+00:00

> _Byline: nim-chat-probe · 2026-08-26_

- `/models` listed 83 models; 75 probed; 20 answered chat; 12 emitted a tool call; 10 accepted an image
- endpoint: `https://integrate.api.nvidia.com/v1`

## WORKING (sorted by chat latency)

| model | chat s | prompt/completion tok | reasoning field | tool call | tools s | vision | vision s | vision reply |
|---|---|---|---|---|---|---|---|---|
| `nvidia/nemotron-3.5-content-safety` | 0.33 | 474/5 |  | ERR 400 | 0.28 | yes | 0.44 | User Safety: safe |
| `meta/llama-3.2-11b-vision-instruct` | 0.37 | 43/3 |  | CALL get_weather | 1.59 | yes | 0.65 | Red. |
| `nvidia/riva-translate-4b-instruct-v2` | 0.38 | 21/9 |  | ERR 400 | 0.3 | no (400) | 0.25 |  |
| `nvidia/riva-translate-4b-instruct-v1.1` | 0.39 | 21/9 |  | ERR 400 | 0.25 | no (400) | 0.27 |  |
| `minimaxai/minimax-m3` | 0.42 | 171/3 |  | ERR 429 | 0.26 | no (429) | 0.27 |  |
| `nvidia/nemotron-3-nano-30b-a3b` | 0.46 | 24/16 | yes | CALL get_weather | 1.12 | no (500) | 0.33 |  |
| `openai/gpt-oss-20b` | 0.48 | 73/16 | yes | CALL get_weather | 1.5 | yes | 0.58 |  |
| `nvidia/llama-3.1-nemotron-safety-guard-8b-v3` | 0.49 | 402/9 |  | ERR 400 | 0.29 | no (400) | 0.31 |  |
| `nvidia/nemotron-3-super-120b-a12b` | 0.63 | 24/16 | yes | CALL get_weather | 2.46 | no (400) | 0.36 |  |
| `nvidia/llama-3.1-nemoguard-8b-content-safety` | 0.79 | 402/8 |  | ERR 400 | 0.38 | no (400) | 0.39 |  |
| `nvidia/nemotron-3.5-lightning-30b-a3b` | 0.86 | 24/16 | yes | CALL get_weather | 1.67 | no (400) | 0.4 |  |
| `nvidia/nemotron-3-nano-omni-30b-a3b-reasoning` | 1.38 | 24/58 | yes | CALL get_weather | 2.29 | yes | 1.37 | Red |
| `nvidia/nemotron-3-ultra-550b-a55b` | 1.46 | 24/16 | yes | ERR 500 | 0.4 | no (400) | 0.4 |  |
| `nvidia/ising-calibration-1.5-31b` | 2.07 | 21/3 |  | text-only | 2.61 | yes | 1.37 | Red |
| `meta/muse-glimmer-30b` | 2.17 | 64/16 | yes | CALL get_weather | 6.98 | yes | 6.36 |  |
| `stepfun-ai/step-3.7-flash` | 2.76 | 20/16 | yes | CALL get_weather | 3.83 | yes | 1.69 |  |
| `poolside/laguna-xs-2.1` | 6.79 | 21/3 |  | CALL get_weather | 0.67 | no (500) | 0.57 |  |
| `moonshotai/kimi-k3` | 8.15 | 94/16 | yes | CALL get_weather | 2.07 | yes | 19.7 |  |
| `meta/llama-3.2-90b-vision-instruct` | 10.3 | 43/3 |  | CALL get_weather | 13.75 | yes | 42.49 | Red. |
| `google/diffusiongemma-26b-a4b-it` | 13.51 | 21/3 |  | CALL get_weather | 0.58 | yes | 1.26 |  |

## NOT WORKING

| model | status | detail |
|---|---|---|
| `01-ai/yi-large` | 404 | Function '23bd454d-b225-49a3-8118-582a62fc51b8': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `adept/fuyu-8b` | 404 | Function 'e598bfc1-b058-41af-869d-556d3c7e1b48': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `ai21labs/jamba-1.5-large-instruct` | 404 | Function '6497fc2b-7ff8-4019-8946-123dccbfc863': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `aisingapore/sea-lion-7b-instruct` | 404 | Function '02f84bf4-c1a1-489b-a9de-ac3e8dcdec14': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `bigcode/starcoder2-15b` | 404 | 404 page not found
 |
| `databricks/dbrx-instruct` | 404 | Function '3d6c2ff8-8bfc-4d10-8fd0-b7337288e869': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `deepseek-ai/deepseek-coder-6.7b-instruct` | 404 | Function 'e503b15c-62b0-4d69-b532-a88f0bfa2656': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `deepseek-ai/deepseek-v4-flash-0731` | 404 | Function id '281478d0-f307-49f4-9e0f-080b63b16c47' version 'null': Specified function in account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' is not found |
| `google/codegemma-1.1-7b` | 404 | Function 'e2d298c5-204e-4213-b921-9f492cc9011b': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `google/codegemma-7b` | 404 | Function '7dfc10a8-3cc4-448e-97c1-2213308dc222': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `google/deplot` | 404 | Function '784a8ca4-ea7d-4c93-bb46-ec027c3fae47': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `google/gemma-2b` | 404 | Function '04174188-f742-4069-9e72-d77c2b77d3cb': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `google/gemma-3-12b-it` | 404 | Function 'ee47df99-c92b-4dc9-b3a7-f3fb0f087b73': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `google/gemma-3-4b-it` | 404 | Function 'c322f327-55a3-4af3-a91f-c757e2b8b135': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `google/gemma-4-31b-it` | -1 | The read operation timed out |
| `google/recurrentgemma-2b` | 404 | Function '2f495340-a99f-4b4b-89bd-1beb003dd896': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `ibm/granite-3.0-3b-a800m-instruct` | 404 | Function '67324577-3f91-4aa6-b750-97468262530d': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `ibm/granite-3.0-8b-instruct` | 404 | Function '5a24a4f0-2d59-46b3-ac65-42307f2633d1': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `ibm/granite-34b-code-instruct` | 404 | Function '4df48b4f-e3c5-4ade-82c7-c06b65e25d18': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `ibm/granite-8b-code-instruct` | 404 | Function 'af7b6f03-f615-4c5f-86c6-388bd35cede0': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `meta/codellama-70b` | 404 | Function 'f6b06895-d073-4714-8bb2-26c09e9f6597': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `meta/llama-guard-4-12b` | -1 | The read operation timed out |
| `meta/llama2-70b` | 404 | Function '2fddadfb-7e76-4c8a-9b82-f7d3fab94471': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `microsoft/kosmos-2` | 404 | Function '6018fed7-f227-48dc-99bc-3fd4264d5037': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `microsoft/phi-3-vision-128k-instruct` | 404 | Function '20f2537e-8593-4eb9-ad40-60eee3bbaa55': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `microsoft/phi-3.5-moe-instruct` | 404 | Function 'e6cab982-62f4-481e-9a7a-3dedb87dbd01': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `mistralai/codestral-22b-instruct-v0.1` | 404 | Function '9a10b012-e6df-46fd-83b2-700dcbc75814': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `mistralai/mistral-7b-instruct-v0.3` | 404 | Function 'cd89bd68-13e3-47a9-861e-9a62e6e14b05': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `mistralai/mistral-large` | 404 | Function '767b5b9a-3f9d-4c1d-86e8-fa861988cee7': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `mistralai/mistral-large-2-instruct` | 404 | Function '7fadd4de-e22a-48e4-90e9-f02ef14a74b9': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `mistralai/mistral-nemotron` | -1 | The read operation timed out |
| `mistralai/mixtral-8x22b-v0.1` | 404 | Function '39655fc1-9ebc-4b24-963e-6915ea6680de': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `moonshotai/kimi-k2.6` | 404 | Function '23d4f03a-b8a6-4adb-a183-7daa083a09cc': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nv-mistralai/mistral-nemo-12b-instruct` | 404 | Function 'f8c05193-d2e2-4f0f-bb4d-7ad70070002b': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/ai-synthetic-video-detector` | 500 | Internal error while making inference request |
| `nvidia/cosmos-reason2-8b` | 404 | Function 'e199b43b-6c62-4a63-9379-f60e1a953236': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/llama-3.1-nemoguard-8b-topic-control` | -1 | The read operation timed out |
| `nvidia/llama-3.1-nemotron-51b-instruct` | 404 | Function '5beba52c-65a9-4f46-8cd9-656689a1b205': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/llama-3.1-nemotron-70b-instruct` | 404 | Function '9b96341b-9791-4db9-a00d-4e43aa192a39': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/llama-3.1-nemotron-ultra-253b-v1` | 404 | Function '84bf12ff-edbd-4435-baea-0fa6a7453d2e': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/llama3-chatqa-1.5-70b` | 404 | Function '46594287-38b9-481c-a37f-baa02f2d3ba1': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/mistral-nemo-minitron-8b-8k-instruct` | 404 | Function '5aa06dd2-0a02-4a5d-be4c-bf88e956965d': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/nemotron-4-340b-instruct` | 404 | Function 'b0fcd392-e905-4ab4-8eb9-aeae95c30b37': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/nemotron-4-340b-reward` | 404 | Function 'c53ee0e9-bad9-4e09-b365-52c9d6b71254': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/nemotron-nano-3-30b-a3b` | 404 | Model not found |
| `nvidia/nemotron-parse` | 400 | Content cannot be a plain string. The model does not support text input. Content cannot be a plain string. The model does not support text input. |
| `nvidia/neva-22b` | 404 | Function 'bc205f8e-1740-40df-8d32-c4321763498a': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/riva-translate-4b-instruct` | 404 | Function 'f35337fa-b4dd-4996-bcba-5476ee01171d': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `nvidia/vila` | 404 | Function '1c8df143-2303-419b-8b28-b4dd82cfe113': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `openai/gpt-oss-120b` | -1 | The read operation timed out |
| `writer/palmyra-creative-122b` | 404 | Function '00bdd0a7-e38f-4423-9007-c4d8730a3f78': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `writer/palmyra-fin-70b-32k` | 404 | Function '316490c6-f1ed-41f9-9da8-3fa9e885653b': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `writer/palmyra-med-70b` | 404 | Function 'aab71274-5281-4941-b0b8-20f339d1fc7e': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `writer/palmyra-med-70b-32k` | 404 | Function 'd6faa974-3591-49a4-963d-97221d074b2e': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |
| `zyphra/zamba2-7b-instruct` | 404 | Function '8378ffb2-51b0-4140-9684-dda1889373e6': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA' |

## Changes vs previous run

- (no previous run / no changes)
