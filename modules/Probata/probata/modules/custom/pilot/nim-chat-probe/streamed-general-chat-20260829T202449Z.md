# NVIDIA NIM general-chat liveness — 2026-08-29T20:24:49+00:00

> _Byline: Codex · streamed plain-chat probe · 2026-08-29_

- Catalog: 83 models; general-chat candidates: 64; excluded specialized endpoints: 19
- Method: SSE streaming; 20s maximum wait for response/first event; once streaming starts, read continues until `[DONE]` with no whole-answer timeout.
- Visible chat response: 11; exact `PONG`: 11; unavailable (404/410): 46; unresolved: 7

## General chat models that answered

| Model | First token | Total | Exact PONG | Finish |
|---|---:|---:|---|---|
| `meta/llama-3.2-11b-vision-instruct` | 0.2s | 0.24s | yes | stop |
| `nvidia/nemotron-3-nano-30b-a3b` | 0.29s | 0.76s | yes | stop |
| `minimaxai/minimax-m3` | 0.31s | 0.37s | yes | stop |
| `openai/gpt-oss-20b` | 0.37s | 0.72s | yes | stop |
| `nvidia/nemotron-3.5-lightning-30b-a3b` | 0.43s | 3.08s | yes | stop |
| `nvidia/nemotron-3-nano-omni-30b-a3b-reasoning` | 0.44s | 2.16s | yes | stop |
| `nvidia/nemotron-3-super-120b-a12b` | 0.86s | 2.04s | yes | stop |
| `openai/gpt-oss-120b` | 0.87s | 1.27s | yes | stop |
| `mistralai/mistral-nemotron` | 0.88s | 0.94s | yes | stop |
| `meta/llama-3.2-90b-vision-instruct` | 1.29s | 1.89s | yes | stop |
| `meta/muse-glimmer-30b` | 2.3s | 14.78s | yes | stop |

## Unavailable or unresolved

| Model | Outcome | Status | Detail |
|---|---|---:|---|
| `01-ai/yi-large` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '23bd454d-b225-49a3-8118-582a62fc51b8': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `adept/fuyu-8b` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'e598bfc1-b058-41af-869d-556d3c7e1b48': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `ai21labs/jamba-1.5-large-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '6497fc2b-7ff8-4019-8946-123dccbfc863': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `aisingapore/sea-lion-7b-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '02f84bf4-c1a1-489b-a9de-ac3e8dcdec14': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `bigcode/starcoder2-15b` | http_error | 404 | 404 page not found
 |
| `databricks/dbrx-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '3d6c2ff8-8bfc-4d10-8fd0-b7337288e869': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `deepseek-ai/deepseek-coder-6.7b-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'e503b15c-62b0-4d69-b532-a88f0bfa2656': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `deepseek-ai/deepseek-v4-flash-0731` | no_headers | - |  |
| `deepseek-ai/deepseek-v4-pro-0813` | no_headers | - |  |
| `google/codegemma-1.1-7b` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'e2d298c5-204e-4213-b921-9f492cc9011b': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `google/codegemma-7b` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '7dfc10a8-3cc4-448e-97c1-2213308dc222': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `google/deplot` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '784a8ca4-ea7d-4c93-bb46-ec027c3fae47': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `google/diffusiongemma-26b-a4b-it` | no_headers | - |  |
| `google/gemma-2b` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '04174188-f742-4069-9e72-d77c2b77d3cb': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `google/gemma-3-12b-it` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'ee47df99-c92b-4dc9-b3a7-f3fb0f087b73': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `google/gemma-3-4b-it` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'c322f327-55a3-4af3-a91f-c757e2b8b135': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `google/gemma-4-31b-it` | no_headers | - |  |
| `google/recurrentgemma-2b` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '2f495340-a99f-4b4b-89bd-1beb003dd896': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `ibm/granite-3.0-3b-a800m-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '67324577-3f91-4aa6-b750-97468262530d': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `ibm/granite-3.0-8b-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '5a24a4f0-2d59-46b3-ac65-42307f2633d1': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `ibm/granite-34b-code-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '4df48b4f-e3c5-4ade-82c7-c06b65e25d18': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `ibm/granite-8b-code-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'af7b6f03-f615-4c5f-86c6-388bd35cede0': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `meta/codellama-70b` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'f6b06895-d073-4714-8bb2-26c09e9f6597': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `meta/llama2-70b` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '2fddadfb-7e76-4c8a-9b82-f7d3fab94471': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `microsoft/kosmos-2` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '6018fed7-f227-48dc-99bc-3fd4264d5037': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `microsoft/phi-3-vision-128k-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '20f2537e-8593-4eb9-ad40-60eee3bbaa55': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `microsoft/phi-3.5-moe-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'e6cab982-62f4-481e-9a7a-3dedb87dbd01': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `mistralai/codestral-22b-instruct-v0.1` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '9a10b012-e6df-46fd-83b2-700dcbc75814': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `mistralai/mistral-7b-instruct-v0.3` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'cd89bd68-13e3-47a9-861e-9a62e6e14b05': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `mistralai/mistral-large` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '767b5b9a-3f9d-4c1d-86e8-fa861988cee7': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `mistralai/mistral-large-2-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '7fadd4de-e22a-48e4-90e9-f02ef14a74b9': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `mistralai/mixtral-8x22b-v0.1` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '39655fc1-9ebc-4b24-963e-6915ea6680de': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `moonshotai/kimi-k2.6` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '23d4f03a-b8a6-4adb-a183-7daa083a09cc': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `moonshotai/kimi-k3` | no_headers | - |  |
| `nv-mistralai/mistral-nemo-12b-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'f8c05193-d2e2-4f0f-bb4d-7ad70070002b': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `nvidia/cosmos-reason2-8b` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'e199b43b-6c62-4a63-9379-f60e1a953236': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `nvidia/llama-3.1-nemotron-51b-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '5beba52c-65a9-4f46-8cd9-656689a1b205': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `nvidia/llama-3.1-nemotron-70b-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '9b96341b-9791-4db9-a00d-4e43aa192a39': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `nvidia/llama-3.1-nemotron-ultra-253b-v1` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '84bf12ff-edbd-4435-baea-0fa6a7453d2e': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `nvidia/llama3-chatqa-1.5-70b` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '46594287-38b9-481c-a37f-baa02f2d3ba1': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `nvidia/mistral-nemo-minitron-8b-8k-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '5aa06dd2-0a02-4a5d-be4c-bf88e956965d': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `nvidia/nemotron-3-ultra-550b-a55b` | no_headers | - |  |
| `nvidia/nemotron-4-340b-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'b0fcd392-e905-4ab4-8eb9-aeae95c30b37': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `nvidia/nemotron-4-340b-reward` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'c53ee0e9-bad9-4e09-b365-52c9d6b71254': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `nvidia/nemotron-nano-3-30b-a3b` | http_error | 404 | {"error":{"message":"Model not found","type":"Not Found","code":404}} |
| `nvidia/neva-22b` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'bc205f8e-1740-40df-8d32-c4321763498a': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `nvidia/vila` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '1c8df143-2303-419b-8b28-b4dd82cfe113': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `poolside/laguna-xs-2.1` | no_first_data_event | 200 |  |
| `writer/palmyra-creative-122b` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '00bdd0a7-e38f-4423-9007-c4d8730a3f78': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `writer/palmyra-fin-70b-32k` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '316490c6-f1ed-41f9-9da8-3fa9e885653b': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `writer/palmyra-med-70b` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'aab71274-5281-4941-b0b8-20f339d1fc7e': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `writer/palmyra-med-70b-32k` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function 'd6faa974-3591-49a4-963d-97221d074b2e': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |
| `zyphra/zamba2-7b-instruct` | http_error | 404 | {"status":404,"title":"Not Found","detail":"Function '8378ffb2-51b0-4140-9684-dda1889373e6': Not found for account 'VSZXQLRN7geBYJXyHdh0E5S8Siwk5-YWGden8rcPknA'"} |

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
